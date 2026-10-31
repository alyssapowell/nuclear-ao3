-- Deleting a work that had tags failed with a foreign key error on
-- work_tag_summaries.
--
-- Deleting the work cascades to its work_tags rows, and each deleted row
-- fires update_work_tag_summary (AFTER DELETE), which re-inserts a summary row
-- for the work. By then the work is gone, so the insert breaks the foreign key
-- from work_tag_summaries to works.
--
-- The summary table stays (the foreign key already has ON DELETE CASCADE, and
-- the tag prominence tests read the table); the trigger now does nothing for a
-- work that no longer exists. tags.use_count needs no change: the work_tags
-- deletes still subtract one per tag through update_tag_use_count, and
-- before_delete_work_tag_counts (migration 033) compensates for hidden works.

CREATE OR REPLACE FUNCTION update_work_tag_summary()
RETURNS TRIGGER AS $$
DECLARE
    target_work UUID := COALESCE(NEW.work_id, OLD.work_id);
BEGIN
    IF NOT EXISTS (SELECT 1 FROM works WHERE id = target_work) THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    INSERT INTO work_tag_summaries (
        work_id,
        primary_relationship_count,
        secondary_relationship_count,
        micro_relationship_count,
        total_tag_count,
        auto_assigned_count,
        last_updated
    ) VALUES (
        target_work,
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id
         WHERE wt.work_id = target_work
         AND wt.prominence = 'primary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id
         WHERE wt.work_id = target_work
         AND wt.prominence = 'secondary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id
         WHERE wt.work_id = target_work
         AND wt.prominence = 'micro' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags WHERE work_id = target_work),
        (SELECT COUNT(*) FROM work_tags WHERE work_id = target_work AND auto_assigned = TRUE),
        CURRENT_TIMESTAMP
    ) ON CONFLICT (work_id) DO UPDATE SET
        primary_relationship_count = EXCLUDED.primary_relationship_count,
        secondary_relationship_count = EXCLUDED.secondary_relationship_count,
        micro_relationship_count = EXCLUDED.micro_relationship_count,
        total_tag_count = EXCLUDED.total_tag_count,
        auto_assigned_count = EXCLUDED.auto_assigned_count,
        last_updated = CURRENT_TIMESTAMP;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- A work's creator can approve or reject (mark spam) a comment on their work,
-- as in AO3, but must not be able to clear a spam mark an administrator set.
-- spam_by_admin records that the mark came from an administrator; the admin
-- spam, unspam and approve actions set and clear it.
ALTER TABLE comments ADD COLUMN spam_by_admin BOOLEAN NOT NULL DEFAULT false;

-- Marks set before this column existed: the audit log knows which were the
-- administrators'.
UPDATE comments c SET spam_by_admin = true
WHERE c.is_spam AND EXISTS (
    SELECT 1 FROM admin_actions a
    WHERE a.target_type = 'comment' AND a.target_id = c.id::text AND a.action = 'spam');
