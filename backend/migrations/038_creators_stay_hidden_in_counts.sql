-- A work that hides its creator (anonymous, in an anonymous collection, or in
-- a collection that hasn't revealed its creators yet) must not show up in
-- anything that counts or lists works by creator, except to the creator.
-- AO3 keeps these works off creator pages and out of creator totals.
--
-- Profile totals come from user_statistics, which is kept by
-- update_user_stats. It counted every posted work; now it skips these.
-- (A creator sees their own totals from live queries, not this table.)

-- True when the work keeps its creators secret.
CREATE OR REPLACE FUNCTION work_hides_creator(work_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false)
           OR coalesce(w.in_unrevealed_collection, false)
    FROM works w WHERE w.id = work_uuid
$$ LANGUAGE sql STABLE;

-- Whether a list of series may show a series to this viewer without giving
-- away who wrote a hidden work in it. A series owner would otherwise be
-- seen next to a work they wrote anonymously. A series shows if it has no works
-- yet, or has one the viewer can see whose creator isn't secret, or the
-- viewer is the creator of the secret work.
CREATE OR REPLACE FUNCTION series_shows_creator(series_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
    SELECT NOT EXISTS (SELECT 1 FROM series_works sw WHERE sw.series_id = series_uuid)
        OR EXISTS (
            SELECT 1 FROM series_works sw JOIN works w ON w.id = sw.work_id
            WHERE sw.series_id = series_uuid
              AND can_user_view_work(w.id, viewer_uuid)
              AND (NOT work_hides_creator(w.id) OR is_work_creator(w.id, viewer_uuid)))
$$ LANGUAGE sql STABLE;

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

-- Totals saved before this change may include such works. Fix them now.
UPDATE user_statistics us SET
    works_count = (
        SELECT COUNT(*) FROM works w
        WHERE w.user_id = us.user_id AND w.is_draft = false AND NOT w.hidden_by_admin AND NOT w.spam
          AND NOT work_hides_creator(w.id)),
    kudos_received_count = (
        SELECT COUNT(*) FROM kudos k JOIN works w ON k.work_id = w.id
        WHERE w.user_id = us.user_id AND NOT work_hides_creator(w.id)),
    words_written = (
        SELECT COALESCE(SUM(w.word_count), 0) FROM works w
        WHERE w.user_id = us.user_id AND w.is_draft = false AND NOT w.hidden_by_admin AND NOT w.spam
          AND NOT work_hides_creator(w.id)),
    last_work_date = (
        SELECT MAX(w.published_at) FROM works w
        WHERE w.user_id = us.user_id AND w.is_draft = false AND NOT w.hidden_by_admin AND NOT w.spam
          AND NOT work_hides_creator(w.id)),
    updated_at = NOW()
WHERE us.user_id IN (SELECT w.user_id FROM works w WHERE work_hides_creator(w.id));
