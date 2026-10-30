-- update_user_stats (from 012) inserted a user_statistics row for the comment's
-- or kudos's user_id, which is NULL for guests, so every guest comment and
-- guest kudos failed.

CREATE OR REPLACE FUNCTION update_user_stats()
RETURNS TRIGGER AS $$
DECLARE
    target_user_id UUID;
BEGIN
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
        ),
        series_count = (
            SELECT COUNT(*) FROM series WHERE user_id = target_user_id
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
        ),
        last_work_date = (
            SELECT MAX(published_at) FROM works 
            WHERE user_id = target_user_id AND is_draft = false
        ),
        updated_at = NOW();
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
