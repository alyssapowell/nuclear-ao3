-- A silently blocked person must not be able to notice the block from a count.
--
-- Migration 042 stores what a person blocked by a hidden creator posts as
-- "shadowed" and leaves it out of every public count. That alone is a leak:
-- they leave kudos or a comment, look at the count, and see it didn't move.
-- So wherever a count is shown to a viewer, it now includes that viewer's own
-- shadowed items on that work. Everyone else keeps seeing the public count.
--
-- These two functions give that extra amount. They return 0 for a guest
-- (NULL viewer), and read a partial index that only holds shadowed rows,
-- which are rare, so the lookup is cheap for everyone else.

CREATE INDEX IF NOT EXISTS idx_kudos_shadowed_by_user
    ON kudos(work_id, user_id) WHERE shadowed;
CREATE INDEX IF NOT EXISTS idx_comments_shadowed_by_user
    ON comments(work_id, user_id) WHERE shadowed;

-- How many kudos the viewer has given on the work that are shadowed.
CREATE OR REPLACE FUNCTION own_shadowed_kudos(work_uuid UUID, viewer_uuid UUID)
RETURNS INTEGER AS $$
    SELECT CASE WHEN viewer_uuid IS NULL THEN 0
        ELSE (SELECT COUNT(*)::int FROM kudos
              WHERE work_id = work_uuid AND user_id = viewer_uuid AND shadowed)
    END
$$ LANGUAGE sql STABLE;

-- How many comments (replies included) the viewer has written on the work
-- that are shadowed and not deleted. Counted the same way as the public
-- comment count in update_work_stats().
CREATE OR REPLACE FUNCTION own_shadowed_comments(work_uuid UUID, viewer_uuid UUID)
RETURNS INTEGER AS $$
    SELECT CASE WHEN viewer_uuid IS NULL THEN 0
        ELSE (SELECT COUNT(*)::int FROM comments
              WHERE work_id = work_uuid AND user_id = viewer_uuid
                AND shadowed AND is_deleted = false)
    END
$$ LANGUAGE sql STABLE;
