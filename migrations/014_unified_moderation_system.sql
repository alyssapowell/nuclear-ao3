-- Migration 014: Unified Moderation System
-- Creates unified reports and moderation_logs tables for admin/moderator tools

-- Create unified reports table (combines work_reports, comment_reports, user_reports)
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('work', 'comment', 'user', 'collection')),
    target_id UUID NOT NULL,
    reporter_id UUID,
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_review', 'resolved', 'dismissed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution TEXT,
    metadata JSONB DEFAULT '{}',
    FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create moderation_logs table for tracking moderator actions
CREATE TABLE IF NOT EXISTS moderation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moderator_id UUID NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('work', 'comment', 'user', 'collection')),
    target_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (moderator_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reports_reporter ON reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_reason ON reports(reason);
CREATE INDEX IF NOT EXISTS idx_reports_created ON reports(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_moderation_logs_moderator ON moderation_logs(moderator_id);
CREATE INDEX IF NOT EXISTS idx_moderation_logs_target ON moderation_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_moderation_logs_action ON moderation_logs(action);
CREATE INDEX IF NOT EXISTS idx_moderation_logs_created ON moderation_logs(created_at DESC);

-- Migrate existing reports from separate tables
INSERT INTO reports (target_type, target_id, reporter_id, reason, description, status, created_at)
SELECT 'work', work_id, reporter_id, reason, description, status, created_at
FROM work_reports
WHERE NOT EXISTS (
    SELECT 1 FROM reports 
    WHERE target_type = 'work' AND target_id = work_reports.work_id 
    AND reporter_id = work_reports.reporter_id AND created_at = work_reports.created_at
);

INSERT INTO reports (target_type, target_id, reporter_id, reason, description, status, created_at)
SELECT 'comment', comment_id, reporter_id, reason, description, status, created_at
FROM comment_reports
WHERE NOT EXISTS (
    SELECT 1 FROM reports 
    WHERE target_type = 'comment' AND target_id = comment_reports.comment_id 
    AND reporter_id = comment_reports.reporter_id AND created_at = comment_reports.created_at
);

-- Add role column to users table if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'role'
    ) THEN
        ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user' 
        CHECK (role IN ('user', 'tag_wrangler', 'moderator', 'admin'));
        CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
    END IF;
END $$;

-- Comments on tables
COMMENT ON TABLE reports IS 'Unified reports table for all content types';
COMMENT ON TABLE moderation_logs IS 'Log of all moderator actions for audit trail';

COMMENT ON COLUMN reports.target_type IS 'Type of content being reported (work, comment, user, collection)';
COMMENT ON COLUMN reports.target_id IS 'ID of the content being reported';
COMMENT ON COLUMN reports.reason IS 'Primary reason for the report';
COMMENT ON COLUMN reports.status IS 'Current status of the report';
COMMENT ON COLUMN reports.metadata IS 'Additional report context and data';

COMMENT ON COLUMN moderation_logs.action IS 'Description of the moderator action taken';
COMMENT ON COLUMN moderation_logs.metadata IS 'Additional context about the moderation action';