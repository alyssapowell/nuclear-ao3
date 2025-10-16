-- OAuth2/OIDC database schema for Nuclear AO3

-- OAuth2 Clients
CREATE TABLE oauth_clients (
    client_id UUID PRIMARY KEY,
    client_secret TEXT, -- Hashed secret for confidential clients
    client_name VARCHAR(100) NOT NULL,
    description TEXT,
    website TEXT,
    logo_url TEXT,
    redirect_uris TEXT[] NOT NULL, -- Array of allowed redirect URIs
    scopes TEXT[] NOT NULL, -- Array of allowed scopes
    grant_types TEXT[] NOT NULL DEFAULT ARRAY['authorization_code'], -- Allowed grant types
    response_types TEXT[] NOT NULL DEFAULT ARRAY['code'], -- Allowed response types
    is_public BOOLEAN NOT NULL DEFAULT false, -- Public vs confidential client
    is_confidential BOOLEAN NOT NULL DEFAULT true, -- Inverse of is_public
    is_trusted BOOLEAN NOT NULL DEFAULT false, -- Skip consent for trusted clients
    is_first_party BOOLEAN NOT NULL DEFAULT false, -- AO3's own applications
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL, -- User who registered the client
    access_token_ttl INTEGER NOT NULL DEFAULT 3600, -- Access token TTL in seconds (1 hour)
    refresh_token_ttl INTEGER NOT NULL DEFAULT 2592000, -- Refresh token TTL in seconds (30 days)
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for oauth_clients
CREATE INDEX idx_oauth_clients_owner_id ON oauth_clients(owner_id);
CREATE INDEX idx_oauth_clients_active ON oauth_clients(is_active);
CREATE INDEX idx_oauth_clients_created_at ON oauth_clients(created_at);

-- Authorization Codes (short-lived)
CREATE TABLE authorization_codes (
    code VARCHAR(255) PRIMARY KEY,
    client_id UUID NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    state TEXT,
    nonce TEXT, -- OIDC nonce
    code_challenge TEXT, -- PKCE code challenge
    code_challenge_method VARCHAR(10), -- PKCE method (S256, plain)
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE, -- NULL if not used yet
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for authorization_codes
CREATE INDEX idx_authorization_codes_client_id ON authorization_codes(client_id);
CREATE INDEX idx_authorization_codes_user_id ON authorization_codes(user_id);
CREATE INDEX idx_authorization_codes_expires_at ON authorization_codes(expires_at);
CREATE INDEX idx_authorization_codes_used_at ON authorization_codes(used_at);

-- OAuth2 Access Tokens
CREATE TABLE oauth_access_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL,
    token_type VARCHAR(50) NOT NULL DEFAULT 'Bearer',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for oauth_access_tokens
CREATE INDEX idx_oauth_access_tokens_token ON oauth_access_tokens(token);
CREATE INDEX idx_oauth_access_tokens_user_id ON oauth_access_tokens(user_id);
CREATE INDEX idx_oauth_access_tokens_client_id ON oauth_access_tokens(client_id);
CREATE INDEX idx_oauth_access_tokens_expires_at ON oauth_access_tokens(expires_at);
CREATE INDEX idx_oauth_access_tokens_revoked ON oauth_access_tokens(is_revoked);
CREATE INDEX idx_oauth_access_tokens_created_at ON oauth_access_tokens(created_at);

-- OAuth2 Refresh Tokens
CREATE TABLE oauth_refresh_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    access_token_id UUID NOT NULL REFERENCES oauth_access_tokens(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for oauth_refresh_tokens
CREATE INDEX idx_oauth_refresh_tokens_token ON oauth_refresh_tokens(token);
CREATE INDEX idx_oauth_refresh_tokens_access_token_id ON oauth_refresh_tokens(access_token_id);
CREATE INDEX idx_oauth_refresh_tokens_user_id ON oauth_refresh_tokens(user_id);
CREATE INDEX idx_oauth_refresh_tokens_client_id ON oauth_refresh_tokens(client_id);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON oauth_refresh_tokens(expires_at);
CREATE INDEX idx_oauth_refresh_tokens_revoked ON oauth_refresh_tokens(is_revoked);

-- User Consents
CREATE TABLE user_consents (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE, -- NULL for permanent consent
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, client_id)
);

-- Indexes for user_consents
CREATE INDEX idx_user_consents_user_id ON user_consents(user_id);
CREATE INDEX idx_user_consents_client_id ON user_consents(client_id);
CREATE INDEX idx_user_consents_revoked ON user_consents(is_revoked);
CREATE INDEX idx_user_consents_expires_at ON user_consents(expires_at);

-- OAuth2 Scopes (for reference and management)
CREATE TABLE oauth_scopes (
    name VARCHAR(100) PRIMARY KEY,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL, -- 'profile', 'works', 'admin', etc.
    is_default BOOLEAN NOT NULL DEFAULT false,
    requires_consent BOOLEAN NOT NULL DEFAULT true,
    is_admin_only BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert default OAuth2 scopes
INSERT INTO oauth_scopes (name, description, category, is_default, requires_consent, is_admin_only) VALUES
('openid', 'OpenID Connect authentication', 'authentication', true, false, false),
('profile', 'Access your basic profile information', 'profile', true, true, false),
('email', 'Access your email address', 'profile', false, true, false),
('read', 'Read your works, bookmarks, and other content', 'works', true, true, false),
('write', 'Create and edit your works', 'works', false, true, false),
('works:manage', 'Full access to manage your works (create, edit, delete)', 'works', false, true, false),
('comments:write', 'Post comments on works', 'social', false, true, false),
('bookmarks:manage', 'Manage your bookmarks', 'works', false, true, false),
('collections:manage', 'Manage collections you own', 'works', false, true, false),
('tags:wrangle', 'Tag wrangling permissions', 'moderation', false, true, true),
('admin', 'Administrative access to the platform', 'admin', false, true, true);

-- OAuth2 Audit Log (for security and compliance)
CREATE TABLE oauth_audit_log (
    id UUID PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL, -- 'token_issued', 'token_revoked', 'consent_granted', etc.
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    client_id UUID REFERENCES oauth_clients(client_id) ON DELETE SET NULL,
    token_id UUID, -- References access or refresh token
    ip_address INET,
    user_agent TEXT,
    details JSONB, -- Additional event-specific data
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for oauth_audit_log
CREATE INDEX idx_oauth_audit_log_event_type ON oauth_audit_log(event_type);
CREATE INDEX idx_oauth_audit_log_user_id ON oauth_audit_log(user_id);
CREATE INDEX idx_oauth_audit_log_client_id ON oauth_audit_log(client_id);
CREATE INDEX idx_oauth_audit_log_created_at ON oauth_audit_log(created_at);
CREATE INDEX idx_oauth_audit_log_details ON oauth_audit_log USING GIN(details);

-- Create first-party AO3 client applications
INSERT INTO oauth_clients (
    client_id, 
    client_name, 
    description, 
    website,
    redirect_uris, 
    scopes, 
    grant_types, 
    response_types,
    is_public,
    is_confidential,
    is_trusted, 
    is_first_party,
    access_token_ttl,
    refresh_token_ttl
) VALUES 
-- AO3 Web Application (main site)
(
    gen_random_uuid(),
    'AO3 Web Application',
    'The main Archive of Our Own web application',
    'https://ao3.example.com',
    ARRAY['https://ao3.example.com/auth/callback'],
    ARRAY['openid', 'profile', 'email', 'read', 'write', 'works:manage', 'comments:write', 'bookmarks:manage', 'collections:manage'],
    ARRAY['authorization_code', 'refresh_token'],
    ARRAY['code'],
    false, -- confidential client
    true,
    true, -- trusted (skip consent)
    true, -- first-party
    3600, -- 1 hour access token
    2592000 -- 30 day refresh token
),
-- AO3 Mobile App
(
    gen_random_uuid(),
    'AO3 Mobile App',
    'Official AO3 mobile application for iOS and Android',
    'https://ao3.example.com/mobile',
    ARRAY['ao3mobile://auth/callback', 'https://ao3.example.com/mobile/auth/callback'],
    ARRAY['openid', 'profile', 'read', 'write', 'works:manage', 'comments:write', 'bookmarks:manage'],
    ARRAY['authorization_code', 'refresh_token'],
    ARRAY['code'],
    true, -- public client (mobile app)
    false,
    true, -- trusted
    true, -- first-party
    3600, -- 1 hour access token
    2592000 -- 30 day refresh token
),
-- AO3 API Documentation/Testing
(
    gen_random_uuid(),
    'AO3 API Explorer',
    'Interactive API documentation and testing tool',
    'https://api-docs.ao3.example.com',
    ARRAY['https://api-docs.ao3.example.com/auth/callback', 'http://localhost:3000/auth/callback'],
    ARRAY['openid', 'profile', 'read'],
    ARRAY['authorization_code', 'refresh_token'],
    ARRAY['code'],
    true, -- public client
    false,
    false, -- requires consent
    true, -- first-party
    3600,
    86400 -- 1 day refresh token
);

-- Update trigger for oauth_clients
CREATE OR REPLACE FUNCTION update_oauth_clients_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_oauth_clients_updated_at 
    BEFORE UPDATE ON oauth_clients 
    FOR EACH ROW EXECUTE FUNCTION update_oauth_clients_updated_at();

-- Cleanup function for expired tokens and codes
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_data()
RETURNS void AS $$
BEGIN
    -- Delete expired authorization codes
    DELETE FROM authorization_codes 
    WHERE expires_at < NOW() - INTERVAL '1 day';
    
    -- Delete expired access tokens
    DELETE FROM oauth_access_tokens 
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    -- Delete expired refresh tokens
    DELETE FROM oauth_refresh_tokens 
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    -- Clean up old audit log entries (keep 1 year)
    DELETE FROM oauth_audit_log 
    WHERE created_at < NOW() - INTERVAL '1 year';
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job to run cleanup (requires pg_cron extension)
-- SELECT cron.schedule('oauth-cleanup', '0 2 * * *', 'SELECT cleanup_expired_oauth_data();');

-- Views for easier querying

-- Active tokens per user
CREATE VIEW user_active_tokens AS
SELECT 
    u.id as user_id,
    u.username,
    c.client_name,
    at.scopes,
    at.created_at,
    at.last_used,
    at.expires_at
FROM oauth_access_tokens at
JOIN users u ON at.user_id = u.id
JOIN oauth_clients c ON at.client_id = c.client_id
WHERE at.is_revoked = false 
AND at.expires_at > NOW()
ORDER BY at.created_at DESC;

-- Client statistics
CREATE VIEW client_statistics AS
SELECT 
    c.client_id,
    c.client_name,
    c.is_public,
    c.is_first_party,
    COUNT(DISTINCT at.user_id) as unique_users,
    COUNT(at.id) as total_tokens_issued,
    COUNT(CASE WHEN at.is_revoked = false AND at.expires_at > NOW() THEN 1 END) as active_tokens,
    MAX(at.created_at) as last_token_issued
FROM oauth_clients c
LEFT JOIN oauth_access_tokens at ON c.client_id = at.client_id
WHERE c.is_active = true
GROUP BY c.client_id, c.client_name, c.is_public, c.is_first_party
ORDER BY unique_users DESC;

-- Add comments for documentation
COMMENT ON TABLE oauth_clients IS 'OAuth2 client applications registered with the system';
COMMENT ON TABLE authorization_codes IS 'Short-lived authorization codes for OAuth2 authorization code flow';
COMMENT ON TABLE oauth_access_tokens IS 'OAuth2 access tokens for API authentication';
COMMENT ON TABLE oauth_refresh_tokens IS 'OAuth2 refresh tokens for obtaining new access tokens';
COMMENT ON TABLE user_consents IS 'User consent records for OAuth2 scopes';
COMMENT ON TABLE oauth_scopes IS 'Available OAuth2 scopes and their properties';
COMMENT ON TABLE oauth_audit_log IS 'Audit trail for OAuth2 security events';

COMMENT ON COLUMN oauth_clients.client_secret IS 'Bcrypt hashed client secret for confidential clients';
COMMENT ON COLUMN oauth_clients.redirect_uris IS 'Array of allowed redirect URIs for this client';
COMMENT ON COLUMN oauth_clients.is_public IS 'Public clients cannot securely store secrets (mobile apps, SPAs)';
COMMENT ON COLUMN oauth_clients.is_trusted IS 'Trusted clients skip the consent screen';
COMMENT ON COLUMN authorization_codes.code_challenge IS 'PKCE code challenge for public clients';
COMMENT ON COLUMN authorization_codes.code_challenge_method IS 'PKCE method: S256 (recommended) or plain';
COMMENT ON COLUMN oauth_access_tokens.scopes IS 'Array of granted scopes for this token';
COMMENT ON COLUMN user_consents.expires_at IS 'NULL means permanent consent until revoked';