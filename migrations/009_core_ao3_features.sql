-- Nuclear AO3: Core AO3 Features Implementation
-- This migration implements anonymous posting, gifting, orphaning, co-authors, and pseudonyms

-- =====================================================
-- PSEUDONYMS SYSTEM
-- =====================================================

-- Create pseuds table (matching AO3's implementation)
CREATE TABLE IF NOT EXISTS pseuds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(40) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, name),
    CONSTRAINT pseud_name_length CHECK (length(name) >= 1 AND length(name) <= 40),
    CONSTRAINT pseud_name_format CHECK (name ~* '^[A-Za-z0-9_.-]+$')
);

-- Indexes for pseuds
CREATE INDEX IF NOT EXISTS idx_pseuds_user ON pseuds(user_id);
CREATE INDEX IF NOT EXISTS idx_pseuds_name ON pseuds(name);
CREATE INDEX IF NOT EXISTS idx_pseuds_default ON pseuds(is_default) WHERE is_default = true;

-- =====================================================
-- CO-AUTHORSHIP SYSTEM
-- =====================================================

-- Create creatorships table (matching AO3's implementation)
CREATE TABLE IF NOT EXISTS creatorships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    creation_id UUID NOT NULL, -- Will reference works, series, etc.
    creation_type VARCHAR(50) NOT NULL, -- 'Work', 'Series', 'Chapter'
    pseud_id UUID NOT NULL REFERENCES pseuds(id) ON DELETE CASCADE,
    approved BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(creation_id, creation_type, pseud_id),
    CONSTRAINT creation_type_values CHECK (creation_type IN ('Work', 'Series', 'Chapter'))
);

-- Indexes for creatorships
CREATE INDEX IF NOT EXISTS idx_creatorships_creation ON creatorships(creation_id, creation_type);
CREATE INDEX IF NOT EXISTS idx_creatorships_pseud ON creatorships(pseud_id);
CREATE INDEX IF NOT EXISTS idx_creatorships_approved ON creatorships(approved);

-- =====================================================
-- GIFTING SYSTEM
-- =====================================================

-- Create gifts table (matching AO3's implementation)
CREATE TABLE IF NOT EXISTS gifts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    pseud_id UUID REFERENCES pseuds(id) ON DELETE SET NULL, -- Recipient's pseud
    recipient_name VARCHAR(100), -- For non-users or additional names
    rejected BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT gift_has_recipient CHECK (pseud_id IS NOT NULL OR recipient_name IS NOT NULL),
    CONSTRAINT recipient_name_format CHECK (recipient_name IS NULL OR recipient_name ~* '[a-zA-Z0-9]'),
    CONSTRAINT recipient_name_length CHECK (recipient_name IS NULL OR length(recipient_name) <= 100)
);

-- Indexes for gifts
CREATE INDEX IF NOT EXISTS idx_gifts_work ON gifts(work_id);
CREATE INDEX IF NOT EXISTS idx_gifts_pseud ON gifts(pseud_id);
CREATE INDEX IF NOT EXISTS idx_gifts_rejected ON gifts(rejected);

-- =====================================================
-- ORPHAN ACCOUNT SYSTEM
-- =====================================================

-- Create orphan account (matching AO3's implementation)
INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'orphan_account',
    'orphan@nuclear-ao3.internal', 
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/VLg7xGx8VYKuOCVla', -- Disabled password
    'orphan_account',
    true,
    true
) ON CONFLICT (username) DO NOTHING;

-- Create default pseud for orphan account
INSERT INTO pseuds (id, user_id, name, is_default)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000000',
    'orphan_account',
    true
) ON CONFLICT (user_id, name) DO NOTHING;

-- =====================================================
-- UPDATE WORKS TABLE FOR NEW FEATURES
-- =====================================================

-- Add anonymous and collection fields to works
ALTER TABLE works ADD COLUMN IF NOT EXISTS in_anon_collection BOOLEAN DEFAULT false;
ALTER TABLE works ADD COLUMN IF NOT EXISTS in_unrevealed_collection BOOLEAN DEFAULT false;
ALTER TABLE works ADD COLUMN IF NOT EXISTS is_anonymous BOOLEAN DEFAULT false;

-- =====================================================
-- CREATE DEFAULT PSEUDS FOR EXISTING USERS
-- =====================================================

-- Create default pseuds for all existing users
INSERT INTO pseuds (user_id, name, is_default)
SELECT id, username, true
FROM users 
WHERE id != '00000000-0000-0000-0000-000000000000'
  AND NOT EXISTS (SELECT 1 FROM pseuds WHERE pseuds.user_id = users.id AND is_default = true);

-- =====================================================
-- CREATE CREATORSHIPS FOR EXISTING WORKS
-- =====================================================

-- Create creatorships for existing works using users' default pseuds
INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved)
SELECT w.id, 'Work', p.id, true
FROM works w
JOIN pseuds p ON w.user_id = p.user_id AND p.is_default = true
WHERE NOT EXISTS (
    SELECT 1 FROM creatorships c 
    WHERE c.creation_id = w.id AND c.creation_type = 'Work'
);

-- =====================================================
-- HELPER FUNCTIONS
-- =====================================================

-- Function to check if work is anonymous
CREATE OR REPLACE FUNCTION is_work_anonymous(work_uuid UUID)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
BEGIN
    SELECT is_anonymous, in_anon_collection
    INTO work_record
    FROM works
    WHERE id = work_uuid;
    
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    RETURN work_record.is_anonymous OR work_record.in_anon_collection;
END;
$$ LANGUAGE plpgsql;

-- Function to get work's visible authors (respecting anonymity)
CREATE OR REPLACE FUNCTION get_work_authors(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS TABLE(
    pseud_id UUID,
    pseud_name VARCHAR,
    user_id UUID,
    username CITEXT,
    is_anonymous BOOLEAN
) AS $$
DECLARE
    is_anon BOOLEAN;
    is_owner BOOLEAN := false;
BEGIN
    -- Check if work is anonymous
    SELECT is_work_anonymous(work_uuid) INTO is_anon;
    
    -- Check if viewer is one of the authors
    IF viewer_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM creatorships c
            JOIN pseuds p ON c.pseud_id = p.id
            WHERE c.creation_id = work_uuid 
            AND c.creation_type = 'Work' 
            AND c.approved = true
            AND p.user_id = viewer_uuid
        ) INTO is_owner;
    END IF;
    
    -- Return author info (anonymous if needed and viewer is not owner)
    RETURN QUERY
    SELECT 
        CASE WHEN is_anon AND NOT is_owner THEN NULL::UUID ELSE p.id END,
        CASE WHEN is_anon AND NOT is_owner THEN 'Anonymous'::VARCHAR ELSE p.name END,
        CASE WHEN is_anon AND NOT is_owner THEN NULL::UUID ELSE p.user_id END,
        CASE WHEN is_anon AND NOT is_owner THEN 'Anonymous'::CITEXT ELSE u.username END,
        is_anon
    FROM creatorships c
    JOIN pseuds p ON c.pseud_id = p.id
    JOIN users u ON p.user_id = u.id
    WHERE c.creation_id = work_uuid 
    AND c.creation_type = 'Work' 
    AND c.approved = true
    ORDER BY c.created_at;
END;
$$ LANGUAGE plpgsql;

-- Function to orphan a work
CREATE OR REPLACE FUNCTION orphan_work(work_uuid UUID, orphaning_user_uuid UUID)
RETURNS BOOLEAN AS $$
DECLARE
    orphan_pseud_id UUID;
    existing_creatorship_count INTEGER;
BEGIN
    -- Get orphan account's default pseud
    SELECT id INTO orphan_pseud_id
    FROM pseuds 
    WHERE user_id = '00000000-0000-0000-0000-000000000000' 
    AND is_default = true;
    
    IF orphan_pseud_id IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check if user is actually an author of this work
    SELECT COUNT(*) INTO existing_creatorship_count
    FROM creatorships c
    JOIN pseuds p ON c.pseud_id = p.id
    WHERE c.creation_id = work_uuid 
    AND c.creation_type = 'Work'
    AND p.user_id = orphaning_user_uuid;
    
    IF existing_creatorship_count = 0 THEN
        RETURN false; -- User is not an author
    END IF;
    
    -- Remove user's creatorships
    DELETE FROM creatorships c
    USING pseuds p
    WHERE c.pseud_id = p.id
    AND c.creation_id = work_uuid 
    AND c.creation_type = 'Work'
    AND p.user_id = orphaning_user_uuid;
    
    -- Add orphan account as author if no other authors remain
    IF NOT EXISTS (
        SELECT 1 FROM creatorships 
        WHERE creation_id = work_uuid AND creation_type = 'Work'
    ) THEN
        INSERT INTO creatorships (creation_id, creation_type, pseud_id, approved)
        VALUES (work_uuid, 'Work', orphan_pseud_id, true);
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's default pseud
CREATE OR REPLACE FUNCTION get_user_default_pseud(user_uuid UUID)
RETURNS UUID AS $$
DECLARE
    pseud_id UUID;
BEGIN
    SELECT id INTO pseud_id
    FROM pseuds
    WHERE user_id = user_uuid AND is_default = true;
    
    RETURN pseud_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create pseud
CREATE OR REPLACE FUNCTION create_pseud(user_uuid UUID, pseud_name VARCHAR, is_default_pseud BOOLEAN DEFAULT false)
RETURNS UUID AS $$
DECLARE
    new_pseud_id UUID;
BEGIN
    -- If setting as default, unset other defaults
    IF is_default_pseud THEN
        UPDATE pseuds SET is_default = false WHERE user_id = user_uuid;
    END IF;
    
    INSERT INTO pseuds (user_id, name, is_default)
    VALUES (user_uuid, pseud_name, is_default_pseud)
    RETURNING id INTO new_pseud_id;
    
    RETURN new_pseud_id;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- UPDATE PRIVACY FUNCTIONS FOR ANONYMOUS WORKS
-- =====================================================

-- Update can_user_view_work to handle anonymous works
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
    is_muted BOOLEAN := false;
    is_author BOOLEAN := false;
BEGIN
    -- Get work privacy settings
    SELECT restricted_to_users, restricted_to_adults, status, user_id, is_anonymous, in_anon_collection
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Check if viewer is one of the authors (for anonymous works)
    IF viewer_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM creatorships c
            JOIN pseuds p ON c.pseud_id = p.id
            WHERE c.creation_id = work_uuid 
            AND c.creation_type = 'Work' 
            AND c.approved = true
            AND p.user_id = viewer_uuid
        ) INTO is_author;
    END IF;
    
    -- Draft works are only visible to their authors
    IF work_record.status = 'draft' THEN
        RETURN is_author;
    END IF;
    
    -- Check if work is restricted to users only
    IF work_record.restricted_to_users = true AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;
    
    -- For anonymous works, we still check blocks/mutes against actual authors
    IF viewer_uuid IS NOT NULL AND NOT is_author THEN
        -- Check if viewer is blocked by any author
        SELECT EXISTS(
            SELECT 1 FROM user_blocks ub
            JOIN pseuds p ON ub.blocker_id = p.user_id
            JOIN creatorships c ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid 
            AND c.creation_type = 'Work'
            AND c.approved = true
            AND ub.blocked_id = viewer_uuid 
            AND ub.block_type IN ('full', 'works')
        ) INTO is_blocked;
        
        IF is_blocked THEN
            RETURN false;
        END IF;
        
        -- Check if viewer has muted any author
        SELECT EXISTS(
            SELECT 1 FROM user_mutes um
            JOIN pseuds p ON um.muted_id = p.user_id
            JOIN creatorships c ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid 
            AND c.creation_type = 'Work'
            AND c.approved = true
            AND um.muter_id = viewer_uuid
        ) INTO is_muted;
        
        IF is_muted THEN
            RETURN false;
        END IF;
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- TRIGGERS FOR MAINTAINING DATA INTEGRITY
-- =====================================================

-- Trigger to ensure users always have a default pseud
CREATE OR REPLACE FUNCTION ensure_default_pseud()
RETURNS TRIGGER AS $$
BEGIN
    -- When a default pseud is deleted, set another one as default
    IF OLD.is_default = true THEN
        UPDATE pseuds 
        SET is_default = true 
        WHERE user_id = OLD.user_id 
        AND id != OLD.id 
        AND id = (
            SELECT id FROM pseuds 
            WHERE user_id = OLD.user_id 
            AND id != OLD.id 
            ORDER BY created_at 
            LIMIT 1
        );
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ensure_default_pseud_trigger
    BEFORE DELETE ON pseuds
    FOR EACH ROW
    EXECUTE FUNCTION ensure_default_pseud();

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE pseuds IS 'User pseudonyms for anonymous/alternate identity posting';
COMMENT ON TABLE creatorships IS 'Links works/series to their creators (pseuds)';
COMMENT ON TABLE gifts IS 'Work gifts to users or named recipients';

COMMENT ON COLUMN works.is_anonymous IS 'Work posted anonymously by author choice';
COMMENT ON COLUMN works.in_anon_collection IS 'Work in an anonymous collection/challenge';
COMMENT ON COLUMN works.in_unrevealed_collection IS 'Work in an unrevealed collection';

COMMENT ON FUNCTION is_work_anonymous(UUID) IS 'Checks if work should display anonymously';
COMMENT ON FUNCTION get_work_authors(UUID, UUID) IS 'Gets work authors respecting anonymity settings';
COMMENT ON FUNCTION orphan_work(UUID, UUID) IS 'Orphans a work to the orphan account';
COMMENT ON FUNCTION get_user_default_pseud(UUID) IS 'Gets a user default pseud ID';
COMMENT ON FUNCTION create_pseud(UUID, VARCHAR, BOOLEAN) IS 'Creates a new pseud for a user';