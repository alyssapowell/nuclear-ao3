-- Add token blacklist for logout functionality
CREATE TABLE token_blacklist (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    blacklisted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist(expires_at);

-- Cleanup job should run periodically: DELETE FROM token_blacklist WHERE expires_at < NOW();
