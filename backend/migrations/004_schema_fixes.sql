-- Nuclear AO3: Schema Fixes and Missing Columns
-- This migration adds columns that were missing during initial service testing
-- Run after 003_oauth2_tables.sql

-- =====================================================
-- WORK TABLE ENHANCEMENTS
-- =====================================================

-- Add missing columns to works table for service compatibility
ALTER TABLE works ADD COLUMN IF NOT EXISTS warnings TEXT;
ALTER TABLE works ADD COLUMN IF NOT EXISTS fandoms TEXT[];
ALTER TABLE works ADD COLUMN IF NOT EXISTS characters TEXT[];
ALTER TABLE works ADD COLUMN IF NOT EXISTS relationships TEXT[];
ALTER TABLE works ADD COLUMN IF NOT EXISTS freeform_tags TEXT[];
ALTER TABLE works ADD COLUMN IF NOT EXISTS max_chapters INTEGER;
ALTER TABLE works ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'published';

-- Add constraint for status values
ALTER TABLE works ADD CONSTRAINT work_status_values 
    CHECK (status IN ('draft', 'published', 'complete', 'abandoned', 'hiatus'));

-- =====================================================
-- WORK STATISTICS TABLE
-- =====================================================

-- Create work_statistics table for detailed analytics
CREATE TABLE IF NOT EXISTS work_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    hits INTEGER DEFAULT 0,
    kudos INTEGER DEFAULT 0,
    comments INTEGER DEFAULT 0,
    bookmarks INTEGER DEFAULT 0,
    hit_count INTEGER DEFAULT 0,
    kudos_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    bookmark_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(work_id)
);

-- Indexes for work_statistics
CREATE INDEX IF NOT EXISTS idx_work_statistics_work_id ON work_statistics(work_id);
CREATE INDEX IF NOT EXISTS idx_work_statistics_hits ON work_statistics(hits DESC);
CREATE INDEX IF NOT EXISTS idx_work_statistics_kudos ON work_statistics(kudos DESC);
CREATE INDEX IF NOT EXISTS idx_work_statistics_updated ON work_statistics(updated_at DESC);

-- =====================================================
-- TAG TABLE ENHANCEMENTS
-- =====================================================

-- Add missing columns to tags table for service compatibility
ALTER TABLE tags ADD COLUMN IF NOT EXISTS canonical BOOLEAN DEFAULT false;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS external BOOLEAN DEFAULT false;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS common BOOLEAN DEFAULT false;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id);

-- Indexes for new tag columns
CREATE INDEX IF NOT EXISTS idx_tags_canonical ON tags(canonical) WHERE canonical = true;
CREATE INDEX IF NOT EXISTS idx_tags_external ON tags(external) WHERE external = true;
CREATE INDEX IF NOT EXISTS idx_tags_common ON tags(common) WHERE common = true;
CREATE INDEX IF NOT EXISTS idx_tags_created_by ON tags(created_by) WHERE created_by IS NOT NULL;

-- =====================================================
-- DATA MIGRATION FOR EXISTING RECORDS
-- =====================================================

-- Initialize work_statistics for existing works
INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks, hit_count, kudos_count, comment_count, bookmark_count)
SELECT 
    w.id,
    COALESCE(w.hit_count, 0),
    COALESCE(w.kudos_count, 0),
    COALESCE(w.comment_count, 0),
    COALESCE(w.bookmark_count, 0),
    COALESCE(w.hit_count, 0),
    COALESCE(w.kudos_count, 0),
    COALESCE(w.comment_count, 0),
    COALESCE(w.bookmark_count, 0)
FROM works w
WHERE NOT EXISTS (SELECT 1 FROM work_statistics ws WHERE ws.work_id = w.id);

-- Update work status based on completion and draft status
UPDATE works SET status = CASE 
    WHEN is_draft = true THEN 'draft'
    WHEN is_complete = true THEN 'complete'
    ELSE 'published'
END WHERE status IS NULL OR status = 'published';

-- Set canonical tags based on is_canonical column
UPDATE tags SET canonical = is_canonical WHERE canonical IS DISTINCT FROM is_canonical;

-- =====================================================
-- TRIGGERS FOR WORK STATISTICS SYNC
-- =====================================================

-- Function to sync work_statistics with works table
CREATE OR REPLACE FUNCTION sync_work_statistics()
RETURNS TRIGGER AS $$
BEGIN
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
$$ language 'plpgsql';

-- Trigger to keep work_statistics in sync
DROP TRIGGER IF EXISTS sync_work_statistics_trigger ON works;
CREATE TRIGGER sync_work_statistics_trigger
    AFTER INSERT OR UPDATE OR DELETE ON works
    FOR EACH ROW EXECUTE FUNCTION sync_work_statistics();

-- =====================================================
-- UPDATED VIEWS
-- =====================================================

-- Update published works view to include new columns
DROP VIEW IF EXISTS published_works_with_stats;
CREATE VIEW published_works_with_stats AS
SELECT 
    w.*,
    u.username as author_username,
    u.display_name as author_display_name,
    ws.hits,
    ws.kudos,
    ws.comments,
    ws.bookmarks,
    COUNT(DISTINCT wt.tag_id) as tag_count,
    ARRAY_AGG(DISTINCT t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL) as tag_names,
    ARRAY_AGG(DISTINCT t.type ORDER BY t.type) FILTER (WHERE t.type IS NOT NULL) as tag_types
FROM works w
JOIN users u ON w.user_id = u.id
LEFT JOIN work_statistics ws ON w.id = ws.work_id
LEFT JOIN work_tags wt ON w.id = wt.work_id
LEFT JOIN tags t ON wt.tag_id = t.id
WHERE w.is_draft = false AND w.published_at IS NOT NULL
GROUP BY w.id, u.username, u.display_name, ws.hits, ws.kudos, ws.comments, ws.bookmarks;

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE work_statistics IS 'Detailed analytics and statistics for works, separate from basic counts in works table';
COMMENT ON COLUMN works.warnings IS 'Content warnings array for the work';
COMMENT ON COLUMN works.fandoms IS 'Fandom tags array for quick filtering';
COMMENT ON COLUMN works.characters IS 'Character tags array for quick filtering';
COMMENT ON COLUMN works.relationships IS 'Relationship tags array for quick filtering';
COMMENT ON COLUMN works.freeform_tags IS 'Freeform tags array for additional categorization';
COMMENT ON COLUMN works.max_chapters IS 'Maximum number of planned chapters, NULL if unknown';
COMMENT ON COLUMN works.status IS 'Current publication status of the work';

COMMENT ON COLUMN tags.canonical IS 'Whether this tag is canonical (duplicate of is_canonical for service compatibility)';
COMMENT ON COLUMN tags.external IS 'Whether this tag was imported from external source';
COMMENT ON COLUMN tags.common IS 'Whether this tag is commonly used and should appear in autocomplete';
COMMENT ON COLUMN tags.created_by IS 'User who created this tag (for tag wrangling audit trail)';

COMMENT ON FUNCTION sync_work_statistics() IS 'Maintains work_statistics table in sync with works table changes';