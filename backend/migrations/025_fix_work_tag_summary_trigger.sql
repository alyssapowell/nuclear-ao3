-- The work_tags trigger from 010 joined on work_tags.tag_name, a column that
-- doesn't exist (work_tags has tag_id), so adding any tag to any work failed.

CREATE OR REPLACE FUNCTION update_work_tag_summary()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or insert work tag summary
    INSERT INTO work_tag_summaries (
        work_id,
        primary_relationship_count,
        secondary_relationship_count,
        micro_relationship_count,
        total_tag_count,
        auto_assigned_count,
        last_updated
    ) VALUES (
        COALESCE(NEW.work_id, OLD.work_id),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id 
         WHERE wt.work_id = COALESCE(NEW.work_id, OLD.work_id) 
         AND wt.prominence = 'primary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id 
         WHERE wt.work_id = COALESCE(NEW.work_id, OLD.work_id) 
         AND wt.prominence = 'secondary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON t.id = wt.tag_id 
         WHERE wt.work_id = COALESCE(NEW.work_id, OLD.work_id) 
         AND wt.prominence = 'micro' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags WHERE work_id = COALESCE(NEW.work_id, OLD.work_id)),
        (SELECT COUNT(*) FROM work_tags WHERE work_id = COALESCE(NEW.work_id, OLD.work_id) AND auto_assigned = TRUE),
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
