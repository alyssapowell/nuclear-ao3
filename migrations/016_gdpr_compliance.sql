-- Migration 016: GDPR Compliance Implementation
-- Adds necessary fields and structures for GDPR compliance

-- Add GDPR consent tracking to users table
ALTER TABLE users 
ADD COLUMN gdpr_consent_date TIMESTAMP,
ADD COLUMN privacy_policy_version VARCHAR(10),
ADD COLUMN cookie_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN analytics_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN marketing_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN data_retention_days INTEGER DEFAULT 2555, -- 7 years default
ADD COLUMN account_deletion_requested BOOLEAN DEFAULT FALSE,
ADD COLUMN account_deletion_date TIMESTAMP,
ADD COLUMN gdpr_data_export_requests INTEGER DEFAULT 0,
ADD COLUMN last_gdpr_export TIMESTAMP;

-- Create GDPR audit log table
CREATE TABLE gdpr_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action_type VARCHAR(50) NOT NULL, -- consent_given, data_exported, data_deleted, etc.
    action_details TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    privacy_policy_version VARCHAR(10),
    legal_basis VARCHAR(50) -- consent, legitimate_interest, etc.
);

-- Create data retention tracking table
CREATE TABLE data_retention_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    user_id UUID REFERENCES users(id),
    data_type VARCHAR(50) NOT NULL,
    retention_period_days INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    deletion_status VARCHAR(20) DEFAULT 'pending', -- pending, scheduled, deleted
    deletion_date TIMESTAMP
);

-- Add GDPR fields to export status table
ALTER TABLE export_status 
ADD COLUMN data_controller VARCHAR(100) DEFAULT 'Nuclear AO3',
ADD COLUMN legal_basis VARCHAR(50) DEFAULT 'consent',
ADD COLUMN retention_period_days INTEGER DEFAULT 30;

-- Create consent management table
CREATE TABLE user_consent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    consent_type VARCHAR(50) NOT NULL, -- essential, analytics, marketing, exports
    consent_given BOOLEAN NOT NULL,
    consent_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    consent_withdrawn_date TIMESTAMP,
    privacy_policy_version VARCHAR(10) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    processing_purpose TEXT,
    data_categories TEXT[] -- email, usage_stats, content, etc.
);

-- Create user data export tracking
CREATE TABLE gdpr_data_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    export_type VARCHAR(50) NOT NULL, -- full_export, works_only, profile_only
    request_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completion_date TIMESTAMP,
    download_date TIMESTAMP,
    export_file_path TEXT,
    export_file_size BIGINT,
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, completed, downloaded, expired
    expires_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP + INTERVAL '30 days'
);

-- Indexes for GDPR performance
CREATE INDEX idx_gdpr_audit_user_id ON gdpr_audit_log(user_id);
CREATE INDEX idx_gdpr_audit_created_at ON gdpr_audit_log(created_at);
CREATE INDEX idx_data_retention_expires_at ON data_retention_tracking(expires_at);
CREATE INDEX idx_data_retention_user_id ON data_retention_tracking(user_id);
CREATE INDEX idx_user_consent_user_id ON user_consent(user_id);
CREATE INDEX idx_user_consent_type ON user_consent(consent_type);
CREATE INDEX idx_gdpr_exports_user_id ON gdpr_data_exports(user_id);
CREATE INDEX idx_gdpr_exports_expires_at ON gdpr_data_exports(expires_at);

-- GDPR compliance functions
CREATE OR REPLACE FUNCTION log_gdpr_action(
    p_user_id UUID,
    p_action_type VARCHAR(50),
    p_action_details TEXT DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_privacy_policy_version VARCHAR(10) DEFAULT '1.0',
    p_legal_basis VARCHAR(50) DEFAULT 'consent'
) RETURNS UUID
LANGUAGE plpgsql
AS $$
DECLARE
    audit_id UUID;
BEGIN
    INSERT INTO gdpr_audit_log (
        user_id, action_type, action_details, ip_address, 
        user_agent, privacy_policy_version, legal_basis
    ) VALUES (
        p_user_id, p_action_type, p_action_details, p_ip_address,
        p_user_agent, p_privacy_policy_version, p_legal_basis
    ) RETURNING id INTO audit_id;
    
    RETURN audit_id;
END;
$$;

-- Function to update user consent
CREATE OR REPLACE FUNCTION update_user_consent(
    p_user_id UUID,
    p_consent_type VARCHAR(50),
    p_consent_given BOOLEAN,
    p_privacy_policy_version VARCHAR(10) DEFAULT '1.0',
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL
) RETURNS UUID
LANGUAGE plpgsql
AS $$
DECLARE
    consent_id UUID;
BEGIN
    -- Insert new consent record
    INSERT INTO user_consent (
        user_id, consent_type, consent_given, privacy_policy_version,
        ip_address, user_agent,
        processing_purpose,
        data_categories
    ) VALUES (
        p_user_id, p_consent_type, p_consent_given, p_privacy_policy_version,
        p_ip_address, p_user_agent,
        CASE p_consent_type
            WHEN 'essential' THEN 'Account management and core platform functionality'
            WHEN 'analytics' THEN 'Anonymized usage statistics to improve the platform'
            WHEN 'marketing' THEN 'Service updates and community communications'
            WHEN 'exports' THEN 'EPUB/MOBI generation and download tracking'
            ELSE 'Platform functionality'
        END,
        CASE p_consent_type
            WHEN 'essential' THEN ARRAY['email', 'account_data', 'content']
            WHEN 'analytics' THEN ARRAY['usage_stats', 'performance_data']
            WHEN 'marketing' THEN ARRAY['email', 'preferences']
            WHEN 'exports' THEN ARRAY['usage_stats', 'download_history']
            ELSE ARRAY['basic_data']
        END
    ) RETURNING id INTO consent_id;
    
    -- Update user table
    CASE p_consent_type
        WHEN 'essential' THEN
            UPDATE users SET gdpr_consent_date = CURRENT_TIMESTAMP WHERE id = p_user_id;
        WHEN 'analytics' THEN
            UPDATE users SET analytics_consent = p_consent_given WHERE id = p_user_id;
        WHEN 'marketing' THEN
            UPDATE users SET marketing_consent = p_consent_given WHERE id = p_user_id;
        WHEN 'cookies' THEN
            UPDATE users SET cookie_consent = p_consent_given WHERE id = p_user_id;
        ELSE
            -- Do nothing for unknown consent types
    END CASE;
    
    -- Log the action
    PERFORM log_gdpr_action(
        p_user_id,
        CASE WHEN p_consent_given THEN 'consent_given' ELSE 'consent_withdrawn' END,
        'Consent updated for: ' || p_consent_type,
        p_ip_address,
        p_user_agent,
        p_privacy_policy_version
    );
    
    RETURN consent_id;
END;
$$;

-- Function to schedule user data deletion
CREATE OR REPLACE FUNCTION schedule_user_deletion(
    p_user_id UUID,
    p_deletion_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP + INTERVAL '30 days'
) RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
BEGIN
    -- Mark user for deletion
    UPDATE users 
    SET account_deletion_requested = TRUE,
        account_deletion_date = p_deletion_date
    WHERE id = p_user_id;
    
    -- Log the action
    PERFORM log_gdpr_action(
        p_user_id,
        'deletion_requested',
        'User requested account deletion on: ' || p_deletion_date::text
    );
    
    -- Schedule data retention tracking for all user data
    INSERT INTO data_retention_tracking (
        table_name, record_id, user_id, data_type, 
        retention_period_days, expires_at
    )
    SELECT 
        'users', p_user_id, p_user_id, 'user_account',
        30, p_deletion_date
    WHERE NOT EXISTS (
        SELECT 1 FROM data_retention_tracking 
        WHERE user_id = p_user_id AND table_name = 'users'
    );
    
    RETURN TRUE;
END;
$$;

-- Auto-cleanup function for expired data
CREATE OR REPLACE FUNCTION cleanup_expired_gdpr_data()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    cleanup_count INTEGER := 0;
BEGIN
    -- Clean up expired GDPR data exports
    DELETE FROM gdpr_data_exports 
    WHERE expires_at < CURRENT_TIMESTAMP 
    AND status IN ('completed', 'downloaded');
    
    GET DIAGNOSTICS cleanup_count = ROW_COUNT;
    
    -- Clean up old audit logs (keep 7 years)
    DELETE FROM gdpr_audit_log 
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '7 years';
    
    -- Process scheduled data deletions
    DELETE FROM users 
    WHERE account_deletion_requested = TRUE 
    AND account_deletion_date <= CURRENT_TIMESTAMP;
    
    RETURN cleanup_count;
END;
$$;

-- Trigger for automatic data retention tracking
CREATE OR REPLACE FUNCTION track_data_retention()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Only track for tables with user data
    IF TG_TABLE_NAME IN ('works', 'chapters', 'comments', 'bookmarks', 'export_status') THEN
        INSERT INTO data_retention_tracking (
            table_name, record_id, user_id, data_type,
            retention_period_days, expires_at
        ) VALUES (
            TG_TABLE_NAME,
            NEW.id,
            COALESCE(NEW.user_id, NEW.author_id),
            'user_content',
            2555, -- 7 years default
            CURRENT_TIMESTAMP + INTERVAL '7 years'
        );
    END IF;
    
    RETURN NEW;
END;
$$;

-- Set up triggers for automatic data retention tracking
CREATE TRIGGER trigger_track_works_retention
    AFTER INSERT ON works
    FOR EACH ROW
    EXECUTE FUNCTION track_data_retention();

CREATE TRIGGER trigger_track_chapters_retention
    AFTER INSERT ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION track_data_retention();

CREATE TRIGGER trigger_track_comments_retention
    AFTER INSERT ON comments
    FOR EACH ROW
    EXECUTE FUNCTION track_data_retention();

-- Add helpful comments
COMMENT ON TABLE gdpr_audit_log IS 'Comprehensive audit trail for all GDPR-related actions and data processing activities';
COMMENT ON TABLE user_consent IS 'Granular consent management for different types of data processing';
COMMENT ON TABLE data_retention_tracking IS 'Automated tracking of data retention periods and deletion schedules';
COMMENT ON TABLE gdpr_data_exports IS 'User data export requests and download tracking';

COMMENT ON FUNCTION log_gdpr_action IS 'Centralized logging function for all GDPR compliance activities';
COMMENT ON FUNCTION update_user_consent IS 'Updates user consent preferences with full audit trail';
COMMENT ON FUNCTION schedule_user_deletion IS 'Schedules user account and data deletion in compliance with GDPR';
COMMENT ON FUNCTION cleanup_expired_gdpr_data IS 'Automated cleanup of expired GDPR data and exports';

-- Insert initial privacy policy version
INSERT INTO system_settings (key, value, description) VALUES 
('privacy_policy_version', '1.0', 'Current privacy policy version for GDPR compliance'),
('gdpr_enabled', 'true', 'GDPR compliance features enabled'),
('data_retention_default_days', '2555', 'Default data retention period (7 years)'),
('gdpr_contact_email', 'privacy@nuclear-ao3.org', 'Contact email for GDPR requests')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;