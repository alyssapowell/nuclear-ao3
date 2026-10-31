-- Moderation state, following AO3.
--
-- AO3 keeps the following (db/schema.rb in otwarchive):
--   hidden_by_admin on works, chapters, series, bookmarks, comments (and
--     external_works, which the rebuild has no table for yet);
--   works.spam and works.spam_checked_at;
--   comments.approved (false means "marked as spam"), unreviewed, iced;
--   users.banned, users.suspended, users.suspended_until.
--
-- Where the rebuild already had a column that means the same thing, it is
-- used instead of adding a second one:
--   comments.is_spam is AO3's "approved = false" (a comment marked as spam).
--   works.moderate_comments is AO3's moderated_commenting_enabled.

-- ---------------------------------------------------------------------
-- Columns
-- ---------------------------------------------------------------------

ALTER TABLE works     ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE works     ADD COLUMN spam            BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE works     ADD COLUMN spam_checked_at TIMESTAMPTZ;
ALTER TABLE chapters  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE series    ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE bookmarks ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE comments  ADD COLUMN hidden_by_admin BOOLEAN NOT NULL DEFAULT false;
-- Waiting for the work's creator to approve it (moderated commenting).
ALTER TABLE comments  ADD COLUMN unreviewed      BOOLEAN NOT NULL DEFAULT false;
-- Frozen: nobody can reply to it.
ALTER TABLE comments  ADD COLUMN iced            BOOLEAN NOT NULL DEFAULT false;

-- banned is permanent. suspended lasts until suspended_until; the rule for
-- "is this account suspended right now" is user_standing() below.
ALTER TABLE users     ADD COLUMN banned          BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users     ADD COLUMN suspended       BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users     ADD COLUMN suspended_until TIMESTAMPTZ;

-- Comments that the old status column already marked.
UPDATE comments SET unreviewed = true WHERE status = 'pending';
UPDATE comments SET is_spam = true WHERE status = 'spam';

-- Partial indexes: these flags are rare, so the indexes stay tiny and serve
-- the moderators' "show me everything hidden" lists and the tag-count triggers.
CREATE INDEX idx_works_hidden      ON works (updated_at DESC)    WHERE hidden_by_admin;
CREATE INDEX idx_works_spam        ON works (updated_at DESC)    WHERE spam;
CREATE INDEX idx_chapters_hidden   ON chapters (work_id)         WHERE hidden_by_admin;
CREATE INDEX idx_series_hidden     ON series (updated_at DESC)   WHERE hidden_by_admin;
CREATE INDEX idx_bookmarks_hidden  ON bookmarks (work_id)        WHERE hidden_by_admin;
CREATE INDEX idx_comments_hidden   ON comments (work_id)         WHERE hidden_by_admin;
CREATE INDEX idx_comments_unreviewed ON comments (work_id)       WHERE unreviewed;
CREATE INDEX idx_comments_iced     ON comments (work_id)         WHERE iced;
CREATE INDEX idx_users_banned      ON users (id)                 WHERE banned;
CREATE INDEX idx_users_suspended   ON users (suspended_until)    WHERE suspended;

-- ---------------------------------------------------------------------
-- Who is who
-- ---------------------------------------------------------------------

-- AO3 admins (Admin records, not Users) see everything. The rebuild's
-- closest equivalents are the admin and moderator roles.
CREATE OR REPLACE FUNCTION is_site_moderator(viewer_uuid uuid)
RETURNS BOOLEAN AS $$
    SELECT viewer_uuid IS NOT NULL AND EXISTS (
        SELECT 1 FROM user_roles
        WHERE user_id = viewer_uuid AND role IN ('admin', 'moderator') AND revoked_at IS NULL)
$$ LANGUAGE sql STABLE;

-- A creator of the work: an approved co-creator, or the user who posted it
-- when the work has no creatorship rows at all.
CREATE OR REPLACE FUNCTION is_work_creator(work_uuid uuid, viewer_uuid uuid)
RETURNS BOOLEAN AS $$
    SELECT viewer_uuid IS NOT NULL AND (
        EXISTS (SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
                WHERE c.creation_id = work_uuid AND c.creation_type = 'Work'
                  AND c.approved AND p.user_id = viewer_uuid)
        OR EXISTS (SELECT 1 FROM works w
                WHERE w.id = work_uuid AND w.user_id = viewer_uuid
                  AND NOT EXISTS (SELECT 1 FROM creatorships c
                                  WHERE c.creation_id = w.id AND c.creation_type = 'Work')))
$$ LANGUAGE sql STABLE;

-- What a banned or suspended account may not do. Returns 'banned',
-- 'suspended' or NULL (in good standing). A suspension whose end time has
-- passed no longer counts, even if nothing has cleared the flag yet.
-- AO3 clears the flag with a daily task; checking the time here means
-- nothing has to run for a suspension to end.
CREATE OR REPLACE FUNCTION user_standing(user_uuid uuid)
RETURNS TEXT AS $$
    SELECT CASE
        WHEN u.banned THEN 'banned'
        WHEN u.suspended AND (u.suspended_until IS NULL OR u.suspended_until > now()) THEN 'suspended'
    END
    FROM users u WHERE u.id = user_uuid
$$ LANGUAGE sql STABLE;

-- ---------------------------------------------------------------------
-- Visibility rules
-- ---------------------------------------------------------------------

-- Works. AO3 Work#visible? (app/models/work.rb:562-570): admins see
-- everything; otherwise a posted, unhidden work is visible to any logged-in
-- user, or to everyone if it is not restricted; a draft or hidden work is
-- visible only to its creators.
--
-- Spam works are treated like hidden ones. In AO3 a spam work is always also
-- hidden (the admin "set spam" action and the spam review queue both set
-- hidden_by_admin), so this only differs for imported data that has spam set
-- and hidden clear.
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
    is_muted BOOLEAN := false;
    is_author BOOLEAN := false;
BEGIN
    SELECT restricted, restricted_to_users, restricted_to_adults, status, user_id,
           is_anonymous, in_anon_collection, hidden_by_admin, spam
    INTO work_record
    FROM works
    WHERE id = work_uuid;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    is_author := is_work_creator(work_uuid, viewer_uuid);

    -- Draft works are only visible to their authors
    IF work_record.status = 'draft' THEN
        RETURN is_author;
    END IF;

    -- Hidden or spam: the creators (who are told it is hidden) and admins
    IF work_record.hidden_by_admin OR work_record.spam THEN
        RETURN is_author OR is_site_moderator(viewer_uuid);
    END IF;

    IF is_author THEN
        RETURN true;
    END IF;

    -- Work restricted to logged-in users
    IF (work_record.restricted OR work_record.restricted_to_users) AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;

    IF viewer_uuid IS NOT NULL THEN
        -- Admins see past blocks and mutes
        IF is_site_moderator(viewer_uuid) THEN
            RETURN true;
        END IF;

        -- Check if viewer is blocked by any author
        SELECT EXISTS(
            SELECT 1 FROM user_blocks ub
            JOIN pseuds p ON ub.blocker_id = p.user_id
            JOIN creatorships c ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid
            AND c.creation_type = 'Work'
            AND c.approved = true
            AND ub.blocked_id = viewer_uuid
            AND ub.block_type IN ('full', 'works')
        ) INTO is_blocked;

        IF is_blocked THEN
            RETURN false;
        END IF;

        -- Check if viewer has muted any author
        SELECT EXISTS(
            SELECT 1 FROM user_mutes um
            JOIN pseuds p ON um.muted_id = p.user_id
            JOIN creatorships c ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid
            AND c.creation_type = 'Work'
            AND c.approved = true
            AND um.muter_id = viewer_uuid
        ) INTO is_muted;

        IF is_muted THEN
            RETURN false;
        END IF;
    END IF;

    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Chapters. AO3 stores chapters.hidden_by_admin but nothing in AO3 sets or
-- reads it, so there is no AO3 rule to copy; a hidden chapter follows the
-- work rule: its work's creators and admins can read it, nobody else.
-- A draft chapter is visible to the work's creators only.
CREATE OR REPLACE FUNCTION can_user_view_chapter(chapter_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    ch RECORD;
BEGIN
    SELECT work_id, is_draft, hidden_by_admin INTO ch FROM chapters WHERE id = chapter_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF NOT can_user_view_work(ch.work_id, viewer_uuid) THEN
        RETURN false;
    END IF;
    IF ch.is_draft THEN
        RETURN is_work_creator(ch.work_id, viewer_uuid);
    END IF;
    IF ch.hidden_by_admin THEN
        RETURN is_work_creator(ch.work_id, viewer_uuid) OR is_site_moderator(viewer_uuid);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Series. AO3 Series#visible? (app/models/series.rb:87-95) has the same
-- shape as the work rule: hidden series are for their creators and admins;
-- restricted series need a login.
CREATE OR REPLACE FUNCTION can_user_view_series(series_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    s RECORD;
BEGIN
    SELECT user_id, restricted, hidden_by_admin INTO s FROM series WHERE id = series_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF viewer_uuid IS NOT NULL AND (
           s.user_id = viewer_uuid
           OR EXISTS (SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
                      WHERE c.creation_id = series_uuid AND c.creation_type = 'Series'
                        AND c.approved AND p.user_id = viewer_uuid)
           OR is_site_moderator(viewer_uuid)) THEN
        RETURN true;
    END IF;
    IF s.hidden_by_admin THEN
        RETURN false;
    END IF;
    IF s.restricted AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Bookmarks. AO3 Bookmark#visible? (app/models/bookmark.rb:148-168): the
-- owner always sees their own; admins see public ones; everyone else sees a
-- bookmark only if it is neither private nor hidden and what it points at is
-- visible to them.
--
-- Unlike AO3, the owner is also held to the work rule: a bookmark of a work
-- the owner cannot see is left out of their list, where AO3 shows it with a
-- placeholder.
CREATE OR REPLACE FUNCTION can_user_view_bookmark(bookmark_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    b RECORD;
BEGIN
    SELECT work_id, user_id, is_private, hidden_by_admin INTO b FROM bookmarks WHERE id = bookmark_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF NOT can_user_view_work(b.work_id, viewer_uuid) THEN
        RETURN false;
    END IF;
    IF viewer_uuid IS NOT NULL AND b.user_id = viewer_uuid THEN
        RETURN true;
    END IF;
    IF b.is_private THEN
        RETURN false;
    END IF;
    IF b.hidden_by_admin THEN
        RETURN is_site_moderator(viewer_uuid);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Comments. AO3 rules:
--   * the work (and chapter) must be visible (comments_controller.rb:65-67,
--     application_controller.rb:513-516);
--   * a comment marked as spam is shown to admins only, and its replies go
--     with it (app/views/comments/_comment_thread.html.erb:4);
--   * an unreviewed comment is visible to the work's creators and admins,
--     and to the person who wrote it (comments_controller.rb:179-185);
--   * a comment hidden by an admin is still part of the thread, but its text
--     is not shown (comments_helper.rb:358); this function says the comment
--     exists for the viewer, and the application blanks the text.
CREATE OR REPLACE FUNCTION can_user_view_comment(comment_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    c RECORD;
    the_work uuid;
BEGIN
    SELECT work_id, chapter_id, parent_comment_id, user_id, is_deleted, is_spam, unreviewed
    INTO c FROM comments WHERE id = comment_uuid;
    IF NOT FOUND OR c.is_deleted THEN
        RETURN false;
    END IF;
    the_work := c.work_id;
    IF c.chapter_id IS NOT NULL THEN
        IF NOT can_user_view_chapter(c.chapter_id, viewer_uuid) THEN
            RETURN false;
        END IF;
        SELECT work_id INTO the_work FROM chapters WHERE id = c.chapter_id;
    ELSIF NOT can_user_view_work(the_work, viewer_uuid) THEN
        RETURN false;
    END IF;
    IF c.is_spam AND NOT is_site_moderator(viewer_uuid) THEN
        RETURN false;
    END IF;
    IF c.unreviewed AND NOT (
           is_work_creator(the_work, viewer_uuid)
           OR is_site_moderator(viewer_uuid)
           OR (viewer_uuid IS NOT NULL AND c.user_id = viewer_uuid)) THEN
        RETURN false;
    END IF;
    IF c.parent_comment_id IS NOT NULL THEN
        RETURN can_user_view_comment(c.parent_comment_id, viewer_uuid);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- ---------------------------------------------------------------------
-- Public counts leave out hidden and spam works
-- ---------------------------------------------------------------------

-- The tags of works that are not hidden or spam. Tag pages and counts read
-- this instead of work_tags.
CREATE VIEW visible_work_tags AS
SELECT wt.*
FROM work_tags wt
JOIN works w ON w.id = wt.work_id
WHERE NOT w.hidden_by_admin AND NOT w.spam;

-- tags.use_count is kept up to date by triggers. A work that is hidden or
-- spam is not counted, so a tag's count is the number of works a stranger
-- could find under it.
CREATE OR REPLACE FUNCTION update_tag_use_count()
RETURNS TRIGGER AS $$
DECLARE
    counted BOOLEAN;
BEGIN
    -- A work that is gone (its tags are being deleted along with it) was
    -- counted: before_delete_work_tag_counts below sees to that.
    SELECT NOT (hidden_by_admin OR spam) INTO counted
    FROM works WHERE id = COALESCE(NEW.work_id, OLD.work_id);
    IF NOT FOUND THEN
        counted := true;
    END IF;

    IF counted THEN
        IF TG_OP = 'INSERT' THEN
            UPDATE tags SET use_count = use_count + 1 WHERE id = NEW.tag_id;
        ELSIF TG_OP = 'DELETE' THEN
            UPDATE tags SET use_count = use_count - 1 WHERE id = OLD.tag_id;
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Hiding or unhiding a work moves its tags' counts up or down by one.
CREATE OR REPLACE FUNCTION adjust_tag_counts_when_hidden()
RETURNS TRIGGER AS $$
DECLARE
    was_hidden BOOLEAN := OLD.hidden_by_admin OR OLD.spam;
    now_hidden BOOLEAN := NEW.hidden_by_admin OR NEW.spam;
BEGIN
    IF was_hidden <> now_hidden THEN
        UPDATE tags SET use_count = use_count + CASE WHEN now_hidden THEN -1 ELSE 1 END
        WHERE id IN (SELECT tag_id FROM work_tags WHERE work_id = NEW.id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER adjust_tag_counts_when_hidden
    AFTER UPDATE OF hidden_by_admin, spam ON works
    FOR EACH ROW EXECUTE FUNCTION adjust_tag_counts_when_hidden();

-- Deleting a hidden work deletes its tag rows, which subtracts one from each
-- count. The work was not counted, so put the one back first.
CREATE OR REPLACE FUNCTION before_delete_work_tag_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.hidden_by_admin OR OLD.spam THEN
        UPDATE tags SET use_count = use_count + 1
        WHERE id IN (SELECT tag_id FROM work_tags WHERE work_id = OLD.id);
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER before_delete_work_tag_counts
    BEFORE DELETE ON works
    FOR EACH ROW EXECUTE FUNCTION before_delete_work_tag_counts();

-- A user's public counts (profile page) leave out hidden and spam works.
CREATE OR REPLACE FUNCTION update_user_stats()
RETURNS TRIGGER AS $$
DECLARE
    target_user_id UUID;
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    -- Determine which user's stats to update
    IF TG_TABLE_NAME = 'works' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'comments' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'bookmarks' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'kudos' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    END IF;

    -- Guest comments and guest kudos have no user to count them against.
    IF target_user_id IS NULL THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    -- Update user statistics
    INSERT INTO user_statistics (user_id, join_date)
    VALUES (target_user_id, (SELECT created_at FROM users WHERE id = target_user_id))
    ON CONFLICT (user_id) DO UPDATE SET
        works_count = (
            SELECT COUNT(*) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
        ),
        series_count = (
            SELECT COUNT(*) FROM series WHERE user_id = target_user_id AND NOT hidden_by_admin
        ),
        bookmarks_count = (
            SELECT COUNT(*) FROM bookmarks WHERE user_id = target_user_id
        ),
        comments_count = (
            SELECT COUNT(*) FROM comments
            WHERE user_id = target_user_id AND is_deleted = false
        ),
        kudos_given_count = (
            SELECT COUNT(*) FROM kudos WHERE user_id = target_user_id
        ),
        kudos_received_count = (
            SELECT COUNT(*) FROM kudos k
            JOIN works w ON k.work_id = w.id
            WHERE w.user_id = target_user_id
        ),
        words_written = (
            SELECT COALESCE(SUM(word_count), 0) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
        ),
        last_work_date = (
            SELECT MAX(published_at) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
        ),
        updated_at = NOW();

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Views that list works or count tag use.
CREATE OR REPLACE VIEW popular_works AS
 SELECT w.id,
    w.title,
    w.summary,
    w.notes,
    w.end_notes,
    w.user_id,
    w.language,
    w.rating,
    w.archive_warning,
    w.category,
    w.word_count,
    w.chapter_count,
    w.expected_chapters,
    w.hit_count,
    w.kudos_count,
    w.comment_count,
    w.bookmark_count,
    w.is_complete,
    w.is_draft,
    w.published_at,
    w.completed_at,
    w.imported_from,
    w.external_id,
    w.restricted,
    w.moderated,
    w.created_at,
    w.updated_at,
    u.username AS author_username,
    u.display_name AS author_display_name,
    (((((w.kudos_count * 2) + (w.comment_count * 3)) + (w.bookmark_count * 4)))::numeric + ((w.hit_count)::numeric * 0.1)) AS popularity_score
   FROM (works w
     JOIN users u ON ((w.user_id = u.id)))
  WHERE ((w.is_draft = false) AND (w.published_at IS NOT NULL)
    AND NOT w.hidden_by_admin AND NOT w.spam)
  ORDER BY (((((w.kudos_count * 2) + (w.comment_count * 3)) + (w.bookmark_count * 4)))::numeric + ((w.hit_count)::numeric * 0.1)) DESC;

CREATE OR REPLACE VIEW published_works_with_stats AS
 SELECT w.id,
    w.title,
    w.summary,
    w.notes,
    w.end_notes,
    w.user_id,
    w.language,
    w.rating,
    w.archive_warning,
    w.category,
    w.word_count,
    w.chapter_count,
    w.expected_chapters,
    w.hit_count,
    w.kudos_count,
    w.comment_count,
    w.bookmark_count,
    w.is_complete,
    w.is_draft,
    w.published_at,
    w.completed_at,
    w.imported_from,
    w.external_id,
    w.restricted,
    w.moderated,
    w.created_at,
    w.updated_at,
    w.warnings,
    w.fandoms,
    w.characters,
    w.relationships,
    w.freeform_tags,
    w.max_chapters,
    w.status,
    u.username AS author_username,
    u.display_name AS author_display_name,
    ws.hits,
    ws.kudos,
    ws.comments,
    ws.bookmarks,
    count(DISTINCT wt.tag_id) AS tag_count,
    array_agg(DISTINCT t.name ORDER BY t.name) FILTER (WHERE (t.name IS NOT NULL)) AS tag_names,
    array_agg(DISTINCT t.type ORDER BY t.type) FILTER (WHERE (t.type IS NOT NULL)) AS tag_types
   FROM ((((works w
     JOIN users u ON ((w.user_id = u.id)))
     LEFT JOIN work_statistics ws ON ((w.id = ws.work_id)))
     LEFT JOIN work_tags wt ON ((w.id = wt.work_id)))
     LEFT JOIN tags t ON ((wt.tag_id = t.id)))
  WHERE ((w.is_draft = false) AND (w.published_at IS NOT NULL)
    AND NOT w.hidden_by_admin AND NOT w.spam)
  GROUP BY w.id, u.username, u.display_name, ws.hits, ws.kudos, ws.comments, ws.bookmarks;

CREATE OR REPLACE VIEW tag_stats AS
 SELECT t.id,
    t.name,
    t.canonical_name,
    t.type,
    t.description,
    t.is_canonical,
    t.is_filterable,
    t.use_count,
    t.created_at,
    t.updated_at,
    count(DISTINCT wt.work_id) FILTER (WHERE (w.id IS NULL OR NOT (w.hidden_by_admin OR w.spam))) AS actual_use_count,
    count(DISTINCT wt.work_id) FILTER (WHERE (w.published_at > (now() - '30 days'::interval)
        AND NOT (w.hidden_by_admin OR w.spam))) AS recent_use_count
   FROM ((tags t
     LEFT JOIN work_tags wt ON ((t.id = wt.tag_id)))
     LEFT JOIN works w ON (((wt.work_id = w.id) AND (w.is_draft = false))))
  GROUP BY t.id;

-- Replies could not be saved: this trigger updated the new reply's
-- thread_level, which fired the same trigger again, and so on until Postgres
-- gave up ("stack depth limit exceeded"). The counts only need work when a
-- comment is added, so the comments branch now runs on INSERT only. Replying
-- to a thread is part of what moderation (frozen threads, hidden comments)
-- has to be able to act on.
CREATE OR REPLACE FUNCTION update_comment_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_TABLE_NAME = 'comment_kudos' THEN
        -- Update comment kudos count
        UPDATE comments SET
            kudos_count = (SELECT COUNT(*) FROM comment_kudos WHERE comment_id = NEW.comment_id)
        WHERE id = NEW.comment_id;
    ELSIF TG_TABLE_NAME = 'comments' AND TG_OP = 'INSERT' THEN
        IF NEW.parent_comment_id IS NOT NULL THEN
            -- Update parent comment reply count
            UPDATE comments SET
                reply_count = (SELECT COUNT(*) FROM comments WHERE parent_comment_id = NEW.parent_comment_id AND is_deleted = false)
            WHERE id = NEW.parent_comment_id;

            -- Update comment thread level
            UPDATE comments SET
                thread_level = (SELECT thread_level + 1 FROM comments WHERE id = NEW.parent_comment_id)
            WHERE id = NEW.id;
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
