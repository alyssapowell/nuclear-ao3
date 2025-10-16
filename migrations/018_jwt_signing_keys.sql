-- JWT Signing Keys Table for Persistent Token Validation
-- This ensures tokens remain valid across service restarts

CREATE TABLE IF NOT EXISTS jwt_signing_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_id VARCHAR(255) NOT NULL UNIQUE,
    key_data TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast lookup of active keys
CREATE INDEX IF NOT EXISTS idx_jwt_signing_keys_active ON jwt_signing_keys(is_active, created_at DESC);

-- Add comment
COMMENT ON TABLE jwt_signing_keys IS 'Stores RSA private keys for JWT signing to maintain token validity across service restarts';