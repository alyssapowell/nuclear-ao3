-- Nuclear AO3: Update Privacy and Moderation Controls
-- This migration updates existing tables for privacy and moderation

-- =====================================================
-- UPDATE COMMENTS TABLE FOR MODERATION
-- =====================================================

-- Add moderation fields to existing comments table
ALTER TABLE comments ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'published';
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderation_reason TEXT;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderated_by UUID REFERENCES users(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS is_anonymous BOOLEAN DEFAULT false;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS ip_address INET;

-- Add constraint for comment status values  
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                   WHERE constraint_name = 'comment_status_values' 
                   AND table_name = 'comments') THEN
        ALTER TABLE comments ADD CONSTRAINT comment_status_values 
            CHECK (status IN ('published', 'pending', 'deleted', 'spam', 'hidden'));
    END IF;
END $$;

-- Update existing comments to have published status
UPDATE comments SET status = 'published' WHERE status IS NULL;

-- =====================================================
-- CREATE COMMENT REPORTS TABLE
-- =====================================================

-- Create comment reports table for harassment reporting
CREATE TABLE IF NOT EXISTS comment_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    reporter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_ip INET,
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    resolution TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT report_reason_values CHECK (reason IN ('spam', 'harassment', 'off_topic', 'inappropriate', 'hate_speech', 'doxxing', 'other')),
    CONSTRAINT report_status_values CHECK (status IN ('pending', 'resolved', 'dismissed', 'escalated'))
);

-- Indexes for comment reports
CREATE INDEX IF NOT EXISTS idx_comment_reports_comment ON comment_reports(comment_id);
CREATE INDEX IF NOT EXISTS idx_comment_reports_reporter ON comment_reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_comment_reports_status ON comment_reports(status);
CREATE INDEX IF NOT EXISTS idx_comment_reports_reason ON comment_reports(reason);

-- =====================================================
-- UPDATE EXISTING FUNCTIONS
-- =====================================================

-- Function to check if a user can view a work
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
BEGIN
    -- Get work privacy settings
    SELECT restricted_to_users, restricted_to_adults, status, user_id
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Draft works are only visible to their authors
    IF work_record.status = 'draft' THEN
        RETURN viewer_uuid IS NOT NULL AND viewer_uuid = work_record.user_id;
    END IF;
    
    -- Check if work is restricted to users only
    IF work_record.restricted_to_users = true AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check if viewer is blocked by author
    IF viewer_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM user_blocks 
            WHERE blocker_id = work_record.user_id 
            AND blocked_id = viewer_uuid 
            AND block_type IN ('full', 'works')
        ) INTO is_blocked;
        
        IF is_blocked THEN
            RETURN false;
        END IF;
    END IF;
    
    -- TODO: Add age verification check for restricted_to_adults
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to check if a user can comment on a work
CREATE OR REPLACE FUNCTION can_user_comment_on_work(work_uuid UUID, commenter_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
BEGIN
    -- Get work comment settings
    SELECT comment_policy, disable_comments, user_id
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Comments disabled
    IF work_record.disable_comments = true OR work_record.comment_policy = 'disabled' THEN
        RETURN false;
    END IF;
    
    -- Users only policy
    IF work_record.comment_policy = 'users_only' AND commenter_uuid IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check if commenter is blocked by author
    IF commenter_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM user_blocks 
            WHERE blocker_id = work_record.user_id 
            AND blocked_id = commenter_uuid 
            AND block_type IN ('full', 'comments')
        ) INTO is_blocked;
        
        IF is_blocked THEN
            RETURN false;
        END IF;
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON COLUMN comments.status IS 'Moderation status: published, pending, deleted, spam, hidden';
COMMENT ON COLUMN comments.moderation_reason IS 'Reason for moderation action';
COMMENT ON COLUMN comments.is_anonymous IS 'Whether this is an anonymous comment';

COMMENT ON TABLE comment_reports IS 'User reports for inappropriate comments';

COMMENT ON FUNCTION can_user_view_work(UUID, UUID) IS 'Checks if a user can view a work based on privacy settings and blocks';
COMMENT ON FUNCTION can_user_comment_on_work(UUID, UUID) IS 'Checks if a user can comment on a work based on comment policy and blocks';