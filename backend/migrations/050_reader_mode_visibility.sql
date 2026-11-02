-- Apps see what any logged-in reader sees, never what a person sees as a creator.
--
-- An app acts for a person, so the database is asked about that person. But
-- the visibility functions let a work's creators (and an owner, and staff)
-- see more: their own drafts, works hidden by moderators, a series they made,
-- their private bookmarks. Handed to an app, those exceptions would show
-- which hidden-creator works the person wrote.
--
-- So each visibility function now takes a third argument, as_reader, that
-- defaults to false (every existing call keeps its meaning). With
-- as_reader = true:
--
--   * the viewer counts as a logged-in member (members-only works show);
--   * blocks and mutes the viewer has still apply, as they do for anyone;
--   * every creator, owner and staff exception is off: no drafts, no hidden
--     works, no private bookmarks, no unreviewed or shadowed comments of
--     the viewer's own, no moderator override.
--
-- A NULL viewer is a guest in either mode.
--
-- The old two-argument versions are dropped first. Left in place next to the
-- new ones, a call with two arguments would match both and Postgres would
-- refuse it as ambiguous.

DROP FUNCTION IF EXISTS can_user_view_work(uuid, uuid);
DROP FUNCTION IF EXISTS can_user_view_chapter(uuid, uuid);
DROP FUNCTION IF EXISTS can_user_view_series(uuid, uuid);
DROP FUNCTION IF EXISTS can_user_view_bookmark(uuid, uuid);
DROP FUNCTION IF EXISTS can_user_view_comment(uuid, uuid);
DROP FUNCTION IF EXISTS series_shows_creator(uuid, uuid);

-- Works. Same as migration 041 when as_reader is false.
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_author BOOLEAN := false;
    is_staff BOOLEAN := false;
BEGIN
    SELECT restricted, restricted_to_users, restricted_to_adults, status, user_id,
           is_anonymous, in_anon_collection, hidden_by_admin, spam
    INTO work_record
    FROM works
    WHERE id = work_uuid;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    IF NOT as_reader THEN
        is_author := is_work_creator(work_uuid, viewer_uuid);
        is_staff := is_site_moderator(viewer_uuid);
    END IF;

    IF work_record.status = 'draft' THEN
        RETURN is_author;
    END IF;

    IF work_record.hidden_by_admin OR work_record.spam THEN
        RETURN is_author OR is_staff;
    END IF;

    IF is_author THEN
        RETURN true;
    END IF;

    IF (work_record.restricted OR work_record.restricted_to_users) AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;

    IF viewer_uuid IS NOT NULL THEN
        IF is_staff THEN
            RETURN true;
        END IF;

        IF NOT work_hides_creator(work_uuid) AND EXISTS (
            SELECT 1 FROM user_mutes um
            JOIN creatorships c ON c.pseud_id = um.muted_pseud_id
            WHERE um.muter_id = viewer_uuid
              AND c.creation_id = work_uuid
              AND c.creation_type = 'Work'
              AND c.approved
        ) THEN
            RETURN false;
        END IF;
    END IF;

    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Chapters. Same as migration 033; as a reader there are no draft or hidden
-- chapters.
CREATE OR REPLACE FUNCTION can_user_view_chapter(chapter_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
DECLARE
    ch RECORD;
BEGIN
    SELECT work_id, is_draft, hidden_by_admin INTO ch FROM chapters WHERE id = chapter_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF NOT can_user_view_work(ch.work_id, viewer_uuid, as_reader) THEN
        RETURN false;
    END IF;
    IF as_reader THEN
        RETURN NOT ch.is_draft AND NOT ch.hidden_by_admin;
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

-- Series. Same as migration 033; as a reader the series' creators and staff
-- get no special view.
CREATE OR REPLACE FUNCTION can_user_view_series(series_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
DECLARE
    s RECORD;
BEGIN
    SELECT user_id, restricted, hidden_by_admin INTO s FROM series WHERE id = series_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF NOT as_reader AND viewer_uuid IS NOT NULL AND (
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

-- Bookmarks. Same as migration 033; as a reader the viewer's own private
-- bookmarks are private like anyone's.
CREATE OR REPLACE FUNCTION can_user_view_bookmark(bookmark_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
DECLARE
    b RECORD;
BEGIN
    SELECT work_id, user_id, is_private, hidden_by_admin INTO b FROM bookmarks WHERE id = bookmark_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF NOT can_user_view_work(b.work_id, viewer_uuid, as_reader) THEN
        RETURN false;
    END IF;
    IF NOT as_reader AND viewer_uuid IS NOT NULL AND b.user_id = viewer_uuid THEN
        RETURN true;
    END IF;
    IF b.is_private THEN
        RETURN false;
    END IF;
    IF b.hidden_by_admin THEN
        RETURN NOT as_reader AND is_site_moderator(viewer_uuid);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Comments. Same as migration 042. As a reader: no moderator view of spam
-- or shadowed comments, no creator view of unreviewed ones, and the viewer's
-- own shadowed and unreviewed comments are hidden like anyone's (showing them
-- would tell an app about a silent block).
CREATE OR REPLACE FUNCTION can_user_view_comment(comment_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
DECLARE
    c RECORD;
    the_work uuid;
    is_staff BOOLEAN := false;
    is_closed BOOLEAN := false;
BEGIN
    SELECT work_id, chapter_id, parent_comment_id, user_id, pseudonym_id, is_deleted, is_spam, unreviewed, shadowed
    INTO c FROM comments WHERE id = comment_uuid;
    IF NOT FOUND OR c.is_deleted THEN
        RETURN false;
    END IF;
    IF NOT as_reader THEN
        is_staff := is_site_moderator(viewer_uuid);
    END IF;
    the_work := c.work_id;
    IF c.chapter_id IS NOT NULL THEN
        IF NOT can_user_view_chapter(c.chapter_id, viewer_uuid, as_reader) THEN
            RETURN false;
        END IF;
        SELECT work_id INTO the_work FROM chapters WHERE id = c.chapter_id;
    ELSIF NOT can_user_view_work(the_work, viewer_uuid, as_reader) THEN
        RETURN false;
    END IF;
    IF c.is_spam AND NOT is_staff THEN
        RETURN false;
    END IF;
    IF c.shadowed AND NOT (is_staff
                           OR (NOT as_reader AND viewer_uuid IS NOT NULL AND c.user_id = viewer_uuid)) THEN
        RETURN false;
    END IF;
    IF viewer_uuid IS NOT NULL THEN
        -- Comments are closed to a reader blocked by a creator whose name is
        -- shown. (blocked_from_work does the same for the other mode.)
        IF as_reader THEN
            is_closed := NOT coalesce(work_hides_creator(the_work), false) AND EXISTS (
                SELECT 1 FROM creatorships cr JOIN pseuds p ON p.id = cr.pseud_id
                WHERE cr.creation_id = the_work AND cr.creation_type = 'Work' AND cr.approved
                  AND accounts_blocked(p.user_id, viewer_uuid));
        ELSE
            is_closed := blocked_from_work(the_work, viewer_uuid);
        END IF;
        IF is_closed THEN
            RETURN false;
        END IF;
        IF NOT is_staff THEN
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
           (NOT as_reader AND is_work_creator(the_work, viewer_uuid))
           OR is_staff
           OR (NOT as_reader AND viewer_uuid IS NOT NULL AND c.user_id = viewer_uuid)) THEN
        RETURN false;
    END IF;
    IF c.parent_comment_id IS NOT NULL THEN
        RETURN can_user_view_comment(c.parent_comment_id, viewer_uuid, as_reader);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Whether a list of series may show a series without giving away who wrote a
-- hidden work in it (migration 038). As a reader, only works the reader can
-- see and whose creator isn't secret count: a creator's own exception is off.
CREATE OR REPLACE FUNCTION series_shows_creator(series_uuid UUID, viewer_uuid UUID DEFAULT NULL, as_reader boolean DEFAULT false)
RETURNS BOOLEAN AS $$
    SELECT NOT EXISTS (SELECT 1 FROM series_works sw WHERE sw.series_id = series_uuid)
        OR EXISTS (
            SELECT 1 FROM series_works sw JOIN works w ON w.id = sw.work_id
            WHERE sw.series_id = series_uuid
              AND can_user_view_work(w.id, viewer_uuid, as_reader)
              AND (NOT work_hides_creator(w.id)
                   OR (NOT as_reader AND is_work_creator(w.id, viewer_uuid))))
$$ LANGUAGE sql STABLE;
