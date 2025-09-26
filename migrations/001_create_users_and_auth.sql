-- Nuclear AO3: Complete User Authentication Schema
-- This migration creates all tables needed for modern JWT-based authentication
-- with proper security, monitoring, and performance optimizations

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext"; -- Case insensitive text for emails/usernames

-- =====================================================
-- USERS AND AUTHENTICATION
-- =====================================================

-- Main users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username CITEXT UNIQUE NOT NULL CHECK (length(username) >= 3 AND length(username) <= 50),
    email CITEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    bio TEXT,
    location VARCHAR(100),
    website VARCHAR(500),
    preferences JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT username_format CHECK (username ~* '^[A-Za-z0-9_-]+$')
);

-- Indexes for users table
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_last_login ON users(last_login_at);

-- User roles table
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'tag_wrangler', 'moderator', 'admin')),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by UUID REFERENCES users(id),
    
    PRIMARY KEY (user_id, role)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role);
CREATE INDEX idx_user_roles_active ON user_roles(user_id, role) WHERE revoked_at IS NULL;

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id) WHERE revoked_at IS NULL AND expires_at > NOW();

-- User sessions for security monitoring
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    user_agent TEXT,
    location VARCHAR(255), -- Geo-location based on IP
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_ip_address ON user_sessions(ip_address);
CREATE INDEX idx_user_sessions_created_at ON user_sessions(created_at);
CREATE INDEX idx_user_sessions_active ON user_sessions(user_id) WHERE is_active = true;

-- Password reset tokens
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- Email verification tokens
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_token_hash ON email_verification_tokens(token_hash);

-- Security events for monitoring suspicious activity
CREATE TABLE security_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) DEFAULT 'low' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    ip_address INET NOT NULL,
    user_agent TEXT,
    location VARCHAR(255),
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_security_events_user_id ON security_events(user_id);
CREATE INDEX idx_security_events_event_type ON security_events(event_type);
CREATE INDEX idx_security_events_severity ON security_events(severity);
CREATE INDEX idx_security_events_created_at ON security_events(created_at);
CREATE INDEX idx_security_events_ip_address ON security_events(ip_address);

-- =====================================================
-- TRIGGERS AND FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for users table
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically assign 'user' role to new users
CREATE OR REPLACE FUNCTION assign_default_role()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_roles (user_id, role) VALUES (NEW.id, 'user');
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to assign default role
CREATE TRIGGER assign_user_default_role
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION assign_default_role();

-- Function to log security events
CREATE OR REPLACE FUNCTION log_security_event(
    p_user_id UUID,
    p_event_type VARCHAR(50),
    p_severity VARCHAR(20),
    p_ip_address INET,
    p_user_agent TEXT,
    p_description TEXT,
    p_metadata JSONB DEFAULT '{}'::jsonb
) RETURNS UUID AS $$
DECLARE
    event_id UUID;
BEGIN
    INSERT INTO security_events (user_id, event_type, severity, ip_address, user_agent, description, metadata)
    VALUES (p_user_id, p_event_type, p_severity, p_ip_address, p_user_agent, p_description, p_metadata)
    RETURNING id INTO event_id;
    
    -- If it's a high severity event, we could trigger alerts here
    IF p_severity IN ('high', 'critical') THEN
        -- Could integrate with alerting system
        RAISE NOTICE 'High severity security event: % for user %', p_event_type, p_user_id;
    END IF;
    
    RETURN event_id;
END;
$$ language 'plpgsql';

-- Function to clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired refresh tokens
    DELETE FROM refresh_tokens WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Delete expired password reset tokens
    DELETE FROM password_reset_tokens WHERE expires_at < NOW();
    
    -- Delete expired email verification tokens  
    DELETE FROM email_verification_tokens WHERE expires_at < NOW();
    
    -- Deactivate old sessions (older than 30 days)
    UPDATE user_sessions 
    SET is_active = false 
    WHERE last_seen < NOW() - INTERVAL '30 days' AND is_active = true;
    
    RETURN deleted_count;
END;
$$ language 'plpgsql';

-- =====================================================
-- VIEWS FOR COMMON QUERIES
-- =====================================================

-- View for active users with their roles
CREATE VIEW active_users_with_roles AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.display_name,
    u.is_verified,
    u.last_login_at,
    u.created_at,
    ARRAY_AGG(ur.role ORDER BY ur.role) FILTER (WHERE ur.role IS NOT NULL AND ur.revoked_at IS NULL) as roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.revoked_at IS NULL
WHERE u.is_active = true
GROUP BY u.id, u.username, u.email, u.display_name, u.is_verified, u.last_login_at, u.created_at;

-- View for user security summary
CREATE VIEW user_security_summary AS
SELECT 
    u.id,
    u.username,
    u.last_login_at,
    COUNT(DISTINCT rt.id) FILTER (WHERE rt.revoked_at IS NULL AND rt.expires_at > NOW()) as active_refresh_tokens,
    COUNT(DISTINCT us.id) FILTER (WHERE us.is_active = true) as active_sessions,
    COUNT(DISTINCT se.id) FILTER (WHERE se.created_at > NOW() - INTERVAL '30 days') as recent_security_events,
    MAX(se.created_at) FILTER (WHERE se.severity IN ('high', 'critical')) as last_high_severity_event
FROM users u
LEFT JOIN refresh_tokens rt ON u.id = rt.user_id
LEFT JOIN user_sessions us ON u.id = us.user_id  
LEFT JOIN security_events se ON u.id = se.user_id
WHERE u.is_active = true
GROUP BY u.id, u.username, u.last_login_at;

-- =====================================================
-- SAMPLE DATA FOR DEVELOPMENT
-- =====================================================

-- Insert sample users (passwords are "password123" hashed with bcrypt cost 12)
INSERT INTO users (id, username, email, password_hash, display_name, is_verified) VALUES
    (
        '123e4567-e89b-12d3-a456-426614174000',
        'testuser',
        'test@nuclear-ao3.com',
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla', -- password123
        'Test User',
        true
    ),
    (
        '123e4567-e89b-12d3-a456-426614174001',
        'author2',
        'author2@nuclear-ao3.com', 
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla', -- password123
        'Famous Author',
        true
    ),
    (
        '123e4567-e89b-12d3-a456-426614174002',
        'tagwrangler',
        'tags@nuclear-ao3.com',
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla', -- password123  
        'Tag Wrangler',
        true
    ),
    (
        '123e4567-e89b-12d3-a456-426614174003',
        'admin',
        'admin@nuclear-ao3.com',
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla', -- password123
        'Site Administrator', 
        true
    )
ON CONFLICT (username) DO NOTHING;

-- Assign special roles (default 'user' role is auto-assigned via trigger)
INSERT INTO user_roles (user_id, role, granted_by) VALUES
    ('123e4567-e89b-12d3-a456-426614174002', 'tag_wrangler', '123e4567-e89b-12d3-a456-426614174003'),
    ('123e4567-e89b-12d3-a456-426614174003', 'admin', '123e4567-e89b-12d3-a456-426614174003')
ON CONFLICT (user_id, role) DO NOTHING;

-- =====================================================
-- PERFORMANCE MONITORING
-- =====================================================

-- Table to store performance metrics
CREATE TABLE auth_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    labels JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_auth_metrics_name_time ON auth_metrics(metric_name, recorded_at);
CREATE INDEX idx_auth_metrics_labels ON auth_metrics USING gin(labels);

-- Function to record metrics
CREATE OR REPLACE FUNCTION record_auth_metric(
    p_metric_name VARCHAR(100),
    p_metric_value NUMERIC,
    p_labels JSONB DEFAULT '{}'::jsonb
) RETURNS VOID AS $$
BEGIN
    INSERT INTO auth_metrics (metric_name, metric_value, labels)
    VALUES (p_metric_name, p_metric_value, p_labels);
END;
$$ language 'plpgsql';

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE users IS 'Main users table with modern security features';
COMMENT ON TABLE user_roles IS 'Role-based access control for users';
COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens with automatic rotation';
COMMENT ON TABLE user_sessions IS 'Security monitoring of user login sessions';
COMMENT ON TABLE security_events IS 'Audit log of security-related events';
COMMENT ON FUNCTION cleanup_expired_tokens() IS 'Maintenance function to clean up expired tokens and sessions';
COMMENT ON VIEW active_users_with_roles IS 'Efficient query for users with their active roles';

-- Set up automatic token cleanup (run this as a cron job)
-- SELECT cron.schedule('cleanup-tokens', '0 2 * * *', 'SELECT cleanup_expired_tokens();');