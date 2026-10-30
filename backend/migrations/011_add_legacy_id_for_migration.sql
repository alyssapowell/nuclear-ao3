-- Migration: Add legacy_id field to works table for AO3 migration support
-- This enables preserving original AO3 numeric IDs while using UUIDs as primary keys

-- Add legacy_id field to works table
ALTER TABLE works ADD COLUMN legacy_id INTEGER UNIQUE;

-- Create index for fast lookups during redirect operations
CREATE INDEX idx_works_legacy_id ON works(legacy_id) WHERE legacy_id IS NOT NULL;

-- Add comment explaining the field
COMMENT ON COLUMN works.legacy_id IS 'Original AO3 numeric ID preserved during migration for URL redirects';

-- Note: The UNIQUE constraint ensures no duplicate legacy IDs
-- The partial index (WHERE legacy_id IS NOT NULL) keeps the index small
-- since most new works won't have legacy IDs