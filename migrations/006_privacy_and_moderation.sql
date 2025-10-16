-- Nuclear AO3: Privacy and Content Moderation Controls
-- This migration adds comprehensive privacy and harassment protection features

-- =====================================================
-- WORK PRIVACY CONTROLS
-- =====================================================

-- Add work-level privacy and access controls
ALTER TABLE works ADD COLUMN IF NOT EXISTS restricted_to_users BOOLEAN DEFAULT false;
ALTER TABLE works ADD COLUMN IF NOT EXISTS restricted_to_adults BOOLEAN DEFAULT false;
ALTER TABLE works ADD COLUMN IF NOT EXISTS comment_policy VARCHAR(20) DEFAULT 'open';
ALTER TABLE works ADD COLUMN IF NOT EXISTS moderate_comments BOOLEAN DEFAULT false;
ALTER TABLE works ADD COLUMN IF NOT EXISTS disable_comments BOOLEAN DEFAULT false;

-- Add constraint for comment policy values
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'work_comment_policy_values') THEN
        ALTER TABLE works ADD CONSTRAINT work_comment_policy_values 
            CHECK (comment_policy IN ('open', 'users_only', 'disabled'));
    END IF;
END $$;

-- =====================================================
-- COMMENT MODERATION SYSTEM
-- =====================================================

-- Add comment moderation fields
ALTER TABLE comments ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'published';
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderation_reason TEXT;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderated_by UUID REFERENCES users(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS is_anonymous BOOLEAN DEFAULT false;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS ip_address INET;

-- Add constraint for comment status values
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'comment_status_values') THEN
        ALTER TABLE comments ADD CONSTRAINT comment_status_values 
            CHECK (status IN ('published', 'pending', 'deleted', 'spam', 'hidden'));
    END IF;
END $$;

-- =====================================================
-- USER BLOCKING AND MUTING SYSTEM
-- =====================================================

-- Create user blocking table for harassment protection
CREATE TABLE IF NOT EXISTS user_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    block_type VARCHAR(20) NOT NULL DEFAULT 'full',
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(blocker_id, blocked_id),
    CONSTRAINT no_self_block CHECK (blocker_id != blocked_id),
    CONSTRAINT block_type_values CHECK (block_type IN ('full', 'comments', 'works'))
);

-- Indexes for user blocks
CREATE INDEX IF NOT EXISTS idx_user_blocks_blocker ON user_blocks(blocker_id);
CREATE INDEX IF NOT EXISTS idx_user_blocks_blocked ON user_blocks(blocked_id);
CREATE INDEX IF NOT EXISTS idx_user_blocks_type ON user_blocks(block_type);

-- =====================================================
-- COMMENT REPORTS SYSTEM
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
-- WORK REPORTS SYSTEM
-- =====================================================

-- Create work reports table for content violations
CREATE TABLE IF NOT EXISTS work_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    reporter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_ip INET,
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    resolution TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT work_report_reason_values CHECK (reason IN ('copyright', 'plagiarism', 'harassment', 'inappropriate_content', 'wrong_rating', 'missing_warnings', 'spam', 'other')),
    CONSTRAINT work_report_status_values CHECK (status IN ('pending', 'resolved', 'dismissed', 'escalated'))
);

-- Indexes for work reports
CREATE INDEX IF NOT EXISTS idx_work_reports_work ON work_reports(work_id);
CREATE INDEX IF NOT EXISTS idx_work_reports_reporter ON work_reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_work_reports_status ON work_reports(status);
CREATE INDEX IF NOT EXISTS idx_work_reports_reason ON work_reports(reason);

-- =====================================================
-- PRIVACY SETTINGS TABLE
-- =====================================================

-- Create user privacy settings table
CREATE TABLE IF NOT EXISTS user_privacy_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Work visibility defaults
    default_work_visibility VARCHAR(20) DEFAULT 'public',
    default_comment_policy VARCHAR(20) DEFAULT 'open',
    default_moderate_comments BOOLEAN DEFAULT false,
    
    -- Personal privacy settings
    hide_email BOOLEAN DEFAULT true,
    hide_profile BOOLEAN DEFAULT false,
    allow_guest_comments BOOLEAN DEFAULT true,
    allow_anonymous_kudos BOOLEAN DEFAULT true,
    
    -- Notification preferences
    notify_comments BOOLEAN DEFAULT true,
    notify_kudos BOOLEAN DEFAULT true,
    notify_bookmarks BOOLEAN DEFAULT false,
    notify_follows BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id),
    CONSTRAINT visibility_values CHECK (default_work_visibility IN ('public', 'users_only', 'private')),
    CONSTRAINT comment_policy_values CHECK (default_comment_policy IN ('open', 'users_only', 'disabled'))
);

-- Index for user privacy settings
CREATE INDEX IF NOT EXISTS idx_user_privacy_settings_user ON user_privacy_settings(user_id);

-- =====================================================
-- CONTENT FILTERING FUNCTIONS
-- =====================================================

-- Function to check if a user can view a work
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
BEGIN
    -- Get work privacy settings
    SELECT restricted_to_users, restricted_to_adults, is_draft, status, user_id
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Draft works are only visible to their authors
    IF work_record.is_draft = true OR work_record.status = 'draft' THEN
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
-- INITIAL DATA MIGRATION
-- =====================================================

-- Create default privacy settings for existing users
INSERT INTO user_privacy_settings (user_id)
SELECT id FROM users 
WHERE NOT EXISTS (SELECT 1 FROM user_privacy_settings WHERE user_id = users.id);

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON COLUMN works.restricted_to_users IS 'Requires user account to view this work';
COMMENT ON COLUMN works.restricted_to_adults IS 'Requires age verification to view this work';
COMMENT ON COLUMN works.comment_policy IS 'Who can comment: open, users_only, disabled';
COMMENT ON COLUMN works.moderate_comments IS 'Whether comments require author approval';
COMMENT ON COLUMN works.disable_comments IS 'Whether comments are completely disabled';

COMMENT ON COLUMN comments.status IS 'Moderation status: published, pending, deleted, spam, hidden';
COMMENT ON COLUMN comments.moderation_reason IS 'Reason for moderation action';
COMMENT ON COLUMN comments.is_anonymous IS 'Whether this is an anonymous comment';

COMMENT ON TABLE user_blocks IS 'User blocking system for harassment protection';
COMMENT ON TABLE comment_reports IS 'User reports for inappropriate comments';
COMMENT ON TABLE work_reports IS 'User reports for content violations';
COMMENT ON TABLE user_privacy_settings IS 'Default privacy settings for users';

COMMENT ON FUNCTION can_user_view_work(UUID, UUID) IS 'Checks if a user can view a work based on privacy settings and blocks';
COMMENT ON FUNCTION can_user_comment_on_work(UUID, UUID) IS 'Checks if a user can comment on a work based on comment policy and blocks';