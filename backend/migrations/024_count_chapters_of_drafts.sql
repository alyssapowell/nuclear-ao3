-- update_work_stats (from 002) set a work's chapter_count to its number of
-- posted chapters. A new draft work has only draft chapters, so the count came
-- out as 0 and the chapter_count > 0 check rejected every new work.
--
-- Now: a draft work counts all its chapters; a posted work counts posted ones.
-- The function also handles deletes (where NEW is NULL) by using OLD.
CREATE OR REPLACE FUNCTION update_work_stats()
RETURNS TRIGGER AS $$
DECLARE
    wid UUID := CASE WHEN TG_OP = 'DELETE' THEN OLD.work_id ELSE NEW.work_id END;
BEGIN
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
$$ LANGUAGE plpgsql;
