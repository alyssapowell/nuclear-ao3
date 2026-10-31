-- User-written HTML is now cleaned when it is saved (internal/html). This
-- migration cleans what was saved before: it queues one background job,
-- 'clean-html', which re-cleans every summary, note, chapter, comment and
-- profile text in batches (see cleanHTML in cmd/nuclear).
--
-- That job rewrites old rows, and several triggers react to an update by
-- bumping works.updated_at (so a work looks freshly edited), recounting, or
-- touching user_statistics. A cleanup is not an edit, so the job sets
-- nuclear.maintenance to 'on' for its own transaction (SET LOCAL), and those
-- triggers do nothing while it is on. The setting is unset in every other
-- connection, so normal behaviour does not change. The function bodies below
-- are the existing ones plus the early return.

CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN NEW;
    END IF;
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

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
            kudos_count = (SELECT COUNT(*) FROM kudos WHERE work_id = wid),
            updated_at = NOW()
        WHERE id = wid;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        UPDATE works SET
            comment_count = (SELECT COUNT(*) FROM comments WHERE work_id = wid AND is_deleted = false),
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

CREATE OR REPLACE FUNCTION update_comment_stats() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_TABLE_NAME = 'comment_kudos' THEN
        -- Update comment kudos count
        UPDATE comments SET
            kudos_count = (SELECT COUNT(*) FROM comment_kudos WHERE comment_id = NEW.comment_id)
        WHERE id = NEW.comment_id;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        -- Update parent comment reply count
        IF NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET
                reply_count = (SELECT COUNT(*) FROM comments WHERE parent_comment_id = NEW.parent_comment_id AND is_deleted = false)
            WHERE id = NEW.parent_comment_id;
        END IF;

        -- Update comment thread level
        IF NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET
                thread_level = (SELECT thread_level + 1 FROM comments WHERE id = NEW.parent_comment_id)
            WHERE id = NEW.id;
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

CREATE OR REPLACE FUNCTION update_user_stats() RETURNS trigger
LANGUAGE plpgsql AS $$
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
$$;

CREATE OR REPLACE FUNCTION sync_work_statistics() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF current_setting('nuclear.maintenance', true) = 'on' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_OP = 'INSERT' THEN
        -- Create work_statistics record for new work
        INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks, hit_count, kudos_count, comment_count, bookmark_count)
        VALUES (NEW.id, 0, 0, 0, 0, 0, 0, 0, 0);
    ELSIF TG_OP = 'UPDATE' THEN
        -- Sync statistics from works to work_statistics
        UPDATE work_statistics SET
            hit_count = COALESCE(NEW.hit_count, 0),
            kudos_count = COALESCE(NEW.kudos_count, 0),
            comment_count = COALESCE(NEW.comment_count, 0),
            bookmark_count = COALESCE(NEW.bookmark_count, 0),
            updated_at = NOW()
        WHERE work_id = NEW.id;
    ELSIF TG_OP = 'DELETE' THEN
        -- work_statistics will be deleted by CASCADE
        RETURN OLD;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

INSERT INTO jobs (kind, payload) VALUES ('clean-html', '{}');
