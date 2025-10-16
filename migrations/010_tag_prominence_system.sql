-- Tag Prominence System Migration
-- Designed for safe batch migration of existing works

-- Add prominence tracking to work_tags junction table
ALTER TABLE work_tags ADD COLUMN prominence VARCHAR(20) DEFAULT 'unassigned';
ALTER TABLE work_tags ADD COLUMN prominence_score DECIMAL(5,3) DEFAULT 0.0;
ALTER TABLE work_tags ADD COLUMN auto_assigned BOOLEAN DEFAULT FALSE;
ALTER TABLE work_tags ADD COLUMN migration_batch INTEGER DEFAULT NULL;
ALTER TABLE work_tags ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE work_tags ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create prominence enum constraint
ALTER TABLE work_tags ADD CONSTRAINT check_prominence 
    CHECK (prominence IN ('primary', 'secondary', 'micro', 'unassigned'));

-- Index for efficient filtering
CREATE INDEX idx_work_tags_prominence ON work_tags(prominence, prominence_score);
CREATE INDEX idx_work_tags_migration ON work_tags(migration_batch, auto_assigned);

-- Migration tracking table
CREATE TABLE tag_migration_batches (
    id SERIAL PRIMARY KEY,
    batch_number INTEGER UNIQUE NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    works_processed INTEGER DEFAULT 0,
    works_total INTEGER DEFAULT 0,
    migration_strategy TEXT,
    status VARCHAR(20) DEFAULT 'pending'
);

-- Tag prominence rules table (for consistent auto-assignment)
CREATE TABLE tag_prominence_rules (
    id SERIAL PRIMARY KEY,
    tag_name VARCHAR(255) NOT NULL,
    tag_type VARCHAR(50) NOT NULL, -- 'relationship', 'character', 'fandom', etc.
    default_prominence VARCHAR(20) NOT NULL,
    min_word_threshold INTEGER DEFAULT 0,
    requires_manual_review BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tag_name) REFERENCES tags(name),
    CONSTRAINT check_rule_prominence 
        CHECK (default_prominence IN ('primary', 'secondary', 'micro'))
);

-- Work-level tag summary for efficient queries
CREATE TABLE work_tag_summaries (
    work_id UUID PRIMARY KEY,
    primary_relationship_count INTEGER DEFAULT 0,
    secondary_relationship_count INTEGER DEFAULT 0,
    micro_relationship_count INTEGER DEFAULT 0,
    total_tag_count INTEGER DEFAULT 0,
    needs_author_review BOOLEAN DEFAULT FALSE,
    auto_assigned_count INTEGER DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
);

-- Tag abuse detection metrics
CREATE TABLE work_tag_metrics (
    work_id UUID PRIMARY KEY,
    tags_per_1k_words DECIMAL(8,2) DEFAULT 0,
    relationship_tag_ratio DECIMAL(5,3) DEFAULT 0,
    unique_fandom_count INTEGER DEFAULT 0,
    potential_tag_spam BOOLEAN DEFAULT FALSE,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_id) REFERENCES works(id) ON DELETE CASCADE
);

-- Insert some common relationship prominence rules
INSERT INTO tag_prominence_rules (tag_name, tag_type, default_prominence, min_word_threshold) VALUES
('Gen', 'relationship', 'primary', 0),
('No Romantic Pairings', 'relationship', 'primary', 0);

-- Create indexes for performance
CREATE INDEX idx_work_tag_summaries_counts ON work_tag_summaries(primary_relationship_count, secondary_relationship_count);
CREATE INDEX idx_work_tag_metrics_spam ON work_tag_metrics(potential_tag_spam, tags_per_1k_words);
CREATE INDEX idx_tag_prominence_rules_lookup ON tag_prominence_rules(tag_name, tag_type);

-- Update trigger to maintain summaries
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
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON wt.tag_name = t.name 
         WHERE wt.work_id = COALESCE(NEW.work_id, OLD.work_id) 
         AND wt.prominence = 'primary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON wt.tag_name = t.name 
         WHERE wt.work_id = COALESCE(NEW.work_id, OLD.work_id) 
         AND wt.prominence = 'secondary' AND t.type = 'relationship'),
        (SELECT COUNT(*) FROM work_tags wt JOIN tags t ON wt.tag_name = t.name 
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

CREATE TRIGGER trigger_update_work_tag_summary
    AFTER INSERT OR UPDATE OR DELETE ON work_tags
    FOR EACH ROW EXECUTE FUNCTION update_work_tag_summary();

-- Comments for future developers
COMMENT ON COLUMN work_tags.prominence IS 'Tag prominence level: primary (main focus), secondary (significant), micro (minor/background), unassigned (needs migration)';
COMMENT ON COLUMN work_tags.prominence_score IS 'Calculated score 0.0-1.0 indicating how prominent this tag is in the work';
COMMENT ON COLUMN work_tags.auto_assigned IS 'Whether prominence was assigned automatically during migration vs manually by author';
COMMENT ON TABLE tag_migration_batches IS 'Tracks batch migration progress for safe rollback and monitoring';
COMMENT ON TABLE work_tag_summaries IS 'Denormalized tag counts for efficient filtering queries';