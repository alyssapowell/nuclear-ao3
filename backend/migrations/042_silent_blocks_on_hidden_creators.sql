-- A block must not unmask a hidden creator, and a gift must not say who
-- blocked you.
--
-- Migration 041 made a block close comments and kudos for the blocked person
-- ("Comments are disabled", 403). On a work that hides its creator (anonymous,
-- in an anonymous collection, or in a collection that hasn't revealed its
-- creators) that answer tells the blocked person that the hidden creator
-- blocked them. So on those works the block is applied silently:
--
--   * the blocked person sees exactly what any other reader sees (the real
--     comment list, the real count, no flag);
--   * what they post there is accepted with the normal answer but stored as
--     "shadowed": only they and the site's moderators can see it, it is left
--     out of every public count and list, and it notifies nobody;
--   * kudos work the same way.
--
-- The same goes for gifts: a gift to someone with a block against one of the
-- work's creators used to be refused (403 gift_blocked), which let anyone
-- probe pseuds to map out who had blocked them. It now answers like any gift
-- but is stored as shadowed, and shown to nobody but the work's creators.
--
-- A block from a creator whose name IS shown (a work that doesn't hide its
-- creators) keeps migration 041's behaviour.

ALTER TABLE comments ADD COLUMN IF NOT EXISTS shadowed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE kudos    ADD COLUMN IF NOT EXISTS shadowed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE gifts    ADD COLUMN IF NOT EXISTS shadowed BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN comments.shadowed IS 'Written by someone blocked by a hidden creator of the work. Visible only to its author and moderators; never counted or announced.';
COMMENT ON COLUMN kudos.shadowed    IS 'Given by someone blocked by a hidden creator of the work. Never listed or counted publicly.';
COMMENT ON COLUMN gifts.shadowed    IS 'Refused because of a block, but answered like any gift. Shown only to the work''s creators.';

CREATE INDEX IF NOT EXISTS idx_comments_shadowed ON comments(work_id) WHERE shadowed;
CREATE INDEX IF NOT EXISTS idx_kudos_shadowed ON kudos(work_id) WHERE shadowed;

-- ---------------------------------------------------------------------
-- Which works close, and which are silent
-- ---------------------------------------------------------------------

-- Same as migration 041, except a work that hides its creators never closes:
-- there the block is applied silently (blocked_silently below).
CREATE OR REPLACE FUNCTION blocked_from_work(work_uuid UUID, viewer_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT viewer_uuid IS NOT NULL
        AND NOT coalesce(work_hides_creator(work_uuid), false)
        AND NOT is_work_creator(work_uuid, viewer_uuid)
        AND NOT is_site_moderator(viewer_uuid)
        AND EXISTS (
            SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid AND c.creation_type = 'Work' AND c.approved
              AND accounts_blocked(p.user_id, viewer_uuid))
$$ LANGUAGE sql STABLE;

-- True when a block applies to the viewer on a work that hides its creators.
-- Nothing is refused and nothing is announced; what the viewer posts is stored
-- as shadowed. Creators and moderators are never subject to it.
CREATE OR REPLACE FUNCTION blocked_silently(work_uuid UUID, viewer_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT viewer_uuid IS NOT NULL
        AND coalesce(work_hides_creator(work_uuid), false)
        AND NOT is_work_creator(work_uuid, viewer_uuid)
        AND NOT is_site_moderator(viewer_uuid)
        AND EXISTS (
            SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid AND c.creation_type = 'Work' AND c.approved
              AND accounts_blocked(p.user_id, viewer_uuid))
$$ LANGUAGE sql STABLE;

-- ---------------------------------------------------------------------
-- Visibility of comments
-- ---------------------------------------------------------------------

-- Same as migration 041, plus: a shadowed comment is visible only to its own
-- author and to moderators.
CREATE OR REPLACE FUNCTION can_user_view_comment(comment_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    c RECORD;
    the_work uuid;
BEGIN
    SELECT work_id, chapter_id, parent_comment_id, user_id, pseudonym_id, is_deleted, is_spam, unreviewed, shadowed
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
    IF c.shadowed AND NOT (is_site_moderator(viewer_uuid)
                           OR (viewer_uuid IS NOT NULL AND c.user_id = viewer_uuid)) THEN
        RETURN false;
    END IF;
    IF viewer_uuid IS NOT NULL THEN
        IF blocked_from_work(the_work, viewer_uuid) THEN
            RETURN false;
        END IF;
        IF NOT is_site_moderator(viewer_uuid) THEN
            IF c.user_id IS NOT NULL AND EXISTS (
                   SELECT 1 FROM user_blocks b
                   WHERE b.blocker_id = viewer_uuid AND b.blocked_id = c.user_id) THEN
                RETURN false;
            END IF;
            IF c.pseudonym_id IS NOT NULL
               AND EXISTS (SELECT 1 FROM user_mutes m
                           WHERE m.muter_id = viewer_uuid AND m.muted_pseud_id = c.pseudonym_id)
               AND NOT (c.user_id IS NOT NULL AND work_hides_creator(the_work)
                        AND is_work_creator(the_work, c.user_id)) THEN
                RETURN false;
            END IF;
        END IF;
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
-- Counts
-- ---------------------------------------------------------------------

-- Same as migration 031's, but shadowed comments and kudos are not counted.
CREATE OR REPLACE FUNCTION update_work_stats() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    wid UUID := CASE WHEN TG_OP = 'DELETE' THEN OLD.work_id ELSE NEW.work_id END;
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_TABLE_NAME = 'chapters' THEN
        UPDATE works w SET
            word_count = (SELECT COALESCE(SUM(c.word_count), 0) FROM chapters c
                          WHERE c.work_id = wid AND (w.is_draft OR NOT c.is_draft)),
            chapter_count = GREATEST(1, (SELECT COUNT(*) FROM chapters c
                          WHERE c.work_id = wid AND (w.is_draft OR NOT c.is_draft))),
            updated_at = NOW()
        WHERE w.id = wid;
    ELSIF TG_TABLE_NAME = 'kudos' THEN
        UPDATE works SET
            kudos_count = (SELECT COUNT(*) FROM kudos WHERE work_id = wid AND NOT shadowed),
            updated_at = NOW()
        WHERE id = wid;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        UPDATE works SET
            comment_count = (SELECT COUNT(*) FROM comments WHERE work_id = wid AND is_deleted = false AND NOT shadowed),
            updated_at = NOW()
        WHERE id = wid;
    ELSIF TG_TABLE_NAME = 'bookmarks' THEN
        UPDATE works SET
            bookmark_count = (SELECT COUNT(*) FROM bookmarks WHERE work_id = wid),
            updated_at = NOW()
        WHERE id = wid;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

-- A comment that becomes shadowed (a blocked person editing their own comment
-- on a work that hides its creator) leaves the public count.
DROP TRIGGER IF EXISTS update_work_stats_comments_shadow ON comments;
CREATE TRIGGER update_work_stats_comments_shadow
    AFTER UPDATE OF shadowed ON comments
    FOR EACH ROW WHEN (OLD.shadowed IS DISTINCT FROM NEW.shadowed)
    EXECUTE FUNCTION update_work_stats();

-- Same as migration 033's, but a shadowed reply isn't counted in its
-- parent's reply count, and a comment that becomes shadowed leaves it. (The
-- UPDATE branch only fires when shadowed changes, so the updates below, which
-- change other columns, don't start it again.)
CREATE OR REPLACE FUNCTION update_comment_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_TABLE_NAME = 'comment_kudos' THEN
        UPDATE comments SET
            kudos_count = (SELECT COUNT(*) FROM comment_kudos WHERE comment_id = NEW.comment_id)
        WHERE id = NEW.comment_id;
    ELSIF TG_TABLE_NAME = 'comments' AND TG_OP = 'INSERT' THEN
        IF NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET
                reply_count = (SELECT COUNT(*) FROM comments
                               WHERE parent_comment_id = NEW.parent_comment_id AND is_deleted = false AND NOT shadowed)
            WHERE id = NEW.parent_comment_id;

            UPDATE comments SET
                thread_level = (SELECT thread_level + 1 FROM comments WHERE id = NEW.parent_comment_id)
            WHERE id = NEW.id;
        END IF;
    ELSIF TG_TABLE_NAME = 'comments' AND TG_OP = 'UPDATE'
          AND OLD.shadowed IS DISTINCT FROM NEW.shadowed
          AND NEW.parent_comment_id IS NOT NULL THEN
        UPDATE comments SET
            reply_count = (SELECT COUNT(*) FROM comments
                           WHERE parent_comment_id = NEW.parent_comment_id AND is_deleted = false AND NOT shadowed)
        WHERE id = NEW.parent_comment_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Same as migration 038's, but kudos received don't include shadowed kudos.
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

    -- Update user statistics. Works that hide their creator are left out of
    -- the work totals, so a profile never shows that its owner wrote them.
    INSERT INTO user_statistics (user_id, join_date)
    VALUES (target_user_id, (SELECT created_at FROM users WHERE id = target_user_id))
    ON CONFLICT (user_id) DO UPDATE SET
        works_count = (
            SELECT COUNT(*) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
              AND NOT (coalesce(is_anonymous, false) OR coalesce(in_anon_collection, false)
                       OR coalesce(in_unrevealed_collection, false))
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
            AND NOT k.shadowed
            WHERE w.user_id = target_user_id
              AND NOT (coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false)
                       OR coalesce(w.in_unrevealed_collection, false))
        ),
        words_written = (
            SELECT COALESCE(SUM(word_count), 0) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
              AND NOT (coalesce(is_anonymous, false) OR coalesce(in_anon_collection, false)
                       OR coalesce(in_unrevealed_collection, false))
        ),
        last_work_date = (
            SELECT MAX(published_at) FROM works
            WHERE user_id = target_user_id AND is_draft = false
              AND NOT hidden_by_admin AND NOT spam
              AND NOT (coalesce(is_anonymous, false) OR coalesce(in_anon_collection, false)
                       OR coalesce(in_unrevealed_collection, false))
        ),
        updated_at = NOW();

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------

-- Same as migration 041's, but a shadowed comment notifies nobody.
CREATE OR REPLACE FUNCTION handle_comment_notifications()
RETURNS TRIGGER AS $$
DECLARE
    work_author_id UUID;
    parent_comment_author_id UUID;
    notification_title TEXT;
    notification_message TEXT;
    work_title TEXT;
    commenter_name TEXT;
BEGIN
    IF NEW.shadowed THEN
        RETURN NEW;
    END IF;

    SELECT w.user_id, w.title INTO work_author_id, work_title
    FROM works w
    WHERE w.id = NEW.work_id;

    -- Who the commenter is called. A creator of a work that hides its
    -- creators is never named, and the username is never used.
    IF NEW.user_id IS NOT NULL AND work_hides_creator(NEW.work_id)
       AND is_work_creator(NEW.work_id, NEW.user_id) THEN
        commenter_name := 'Anonymous Creator';
    ELSE
        SELECT p.name INTO commenter_name FROM pseuds p WHERE p.id = NEW.pseudonym_id;
        commenter_name := COALESCE(commenter_name, NEW.guest_name, 'Someone');
    END IF;

    -- Notify the work's author (if not commenting on their own work, and the
    -- author hasn't blocked the commenter).
    IF work_author_id IS NOT NULL AND work_author_id IS DISTINCT FROM NEW.user_id
       AND NOT (NEW.user_id IS NOT NULL AND EXISTS (
            SELECT 1 FROM user_blocks b
            WHERE b.blocker_id = work_author_id AND b.blocked_id = NEW.user_id)) THEN
        notification_title := format('New comment on "%s"', work_title);
        notification_message := format('%s commented on your work', commenter_name);

        PERFORM create_notification(
            work_author_id,
            'work_comment',
            notification_title,
            notification_message,
            jsonb_build_object(
                'work_id', NEW.work_id,
                'comment_id', NEW.id,
                'commenter_name', commenter_name,
                'work_title', work_title
            )
        );
    END IF;

    -- Notify the author of the comment being replied to.
    IF NEW.parent_comment_id IS NOT NULL THEN
        SELECT c.user_id INTO parent_comment_author_id
        FROM comments c
        WHERE c.id = NEW.parent_comment_id;

        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id IS DISTINCT FROM NEW.user_id
           AND NOT (NEW.user_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM user_blocks b
                WHERE b.blocker_id = parent_comment_author_id AND b.blocked_id = NEW.user_id)) THEN
            notification_title := 'Reply to your comment';
            notification_message := format('%s replied to your comment', commenter_name);

            PERFORM create_notification(
                parent_comment_author_id,
                'comment_reply',
                notification_title,
                notification_message,
                jsonb_build_object(
                    'work_id', NEW.work_id,
                    'comment_id', NEW.id,
                    'parent_comment_id', NEW.parent_comment_id,
                    'commenter_name', commenter_name,
                    'work_title', work_title
                )
            );
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
