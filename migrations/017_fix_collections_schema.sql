-- Fix collections table schema
-- Add missing 'name' column that tests expect

ALTER TABLE collections 
ADD COLUMN IF NOT EXISTS name VARCHAR(255) UNIQUE;

-- Populate name column with a URL-friendly version of title for existing records
UPDATE collections 
SET name = LOWER(REPLACE(REPLACE(title, ' ', '-'), '''', ''))
WHERE name IS NULL;

-- Make name NOT NULL after populating
ALTER TABLE collections 
ALTER COLUMN name SET NOT NULL;

-- Add index for performance
CREATE INDEX IF NOT EXISTS idx_collections_name ON collections(name);