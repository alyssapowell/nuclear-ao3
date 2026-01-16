-- Add missing columns to existing refresh_tokens table
-- The table was created in migration 001, but missing user_agent and ip_address columns

-- Add user_agent column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'refresh_tokens' AND column_name = 'user_agent'
    ) THEN
        ALTER TABLE refresh_tokens ADD COLUMN user_agent TEXT;
    END IF;
END $$;

-- Add ip_address column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'refresh_tokens' AND column_name = 'ip_address'
    ) THEN
        ALTER TABLE refresh_tokens ADD COLUMN ip_address TEXT;
    END IF;
END $$;
