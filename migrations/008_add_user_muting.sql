-- Nuclear AO3: User Muting System
-- This migration adds user muting functionality matching AO3's implementation

-- =====================================================
-- USER MUTING TABLE
-- =====================================================

-- Create user mutes table (matching AO3's structure)
CREATE TABLE IF NOT EXISTS user_mutes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    muter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(muter_id, muted_id),
    CONSTRAINT no_self_mute CHECK (muter_id != muted_id)
);

-- Indexes for user mutes
CREATE INDEX IF NOT EXISTS idx_user_mutes_muter ON user_mutes(muter_id);
CREATE INDEX IF NOT EXISTS idx_user_mutes_muted ON user_mutes(muted_id);
CREATE INDEX IF NOT EXISTS idx_user_mutes_created ON user_mutes(created_at);

-- =====================================================
-- UPDATE PRIVACY FUNCTIONS FOR MUTING
-- =====================================================

-- Function to check if a user is muted by another user
CREATE OR REPLACE FUNCTION is_user_muted(muter_uuid UUID, muted_uuid UUID)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if muter has muted the other user
    RETURN EXISTS(
        SELECT 1 FROM user_mutes 
        WHERE muter_id = muter_uuid AND muted_id = muted_uuid
    );
END;
$$ LANGUAGE plpgsql;

-- Update can_user_view_work to respect muting (muted users' works are hidden)
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
    is_muted BOOLEAN := false;
BEGIN
    -- Get work privacy settings
    SELECT restricted_to_users, restricted_to_adults, status, user_id
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Draft works are only visible to their authors
    IF work_record.status = 'draft' THEN
        RETURN viewer_uuid IS NOT NULL AND viewer_uuid = work_record.user_id;
    END IF;
    
    -- Check if work is restricted to users only
    IF work_record.restricted_to_users = true AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check if viewer is blocked by author
    IF viewer_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM user_blocks 
            WHERE blocker_id = work_record.user_id 
            AND blocked_id = viewer_uuid 
            AND block_type IN ('full', 'works')
        ) INTO is_blocked;
        
        IF is_blocked THEN
            RETURN false;
        END IF;
        
        -- Check if viewer has muted the author (hide muted users' works)
        SELECT is_user_muted(viewer_uuid, work_record.user_id) INTO is_muted;
        
        IF is_muted THEN
            RETURN false;
        END IF;
    END IF;
    
    -- TODO: Add age verification check for restricted_to_adults
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Update can_user_comment_on_work to respect muting
CREATE OR REPLACE FUNCTION can_user_comment_on_work(work_uuid UUID, commenter_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
    is_muted_by_author BOOLEAN := false;
    is_muted_by_commenter BOOLEAN := false;
BEGIN
    -- Get work comment settings
    SELECT comment_policy, disable_comments, user_id
    INTO work_record
    FROM works 
    WHERE id = work_uuid;
    
    -- Work doesn't exist
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    
    -- Comments disabled
    IF work_record.disable_comments = true OR work_record.comment_policy = 'disabled' THEN
        RETURN false;
    END IF;
    
    -- Users only policy
    IF work_record.comment_policy = 'users_only' AND commenter_uuid IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check if commenter is blocked by author
    IF commenter_uuid IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1 FROM user_blocks 
            WHERE blocker_id = work_record.user_id 
            AND blocked_id = commenter_uuid 
            AND block_type IN ('full', 'comments')
        ) INTO is_blocked;
        
        IF is_blocked THEN
            RETURN false;
        END IF;
        
        -- Check if author has muted the commenter
        SELECT is_user_muted(work_record.user_id, commenter_uuid) INTO is_muted_by_author;
        
        IF is_muted_by_author THEN
            RETURN false;
        END IF;
        
        -- Check if commenter has muted the author (they can't comment on muted users' works)
        SELECT is_user_muted(commenter_uuid, work_record.user_id) INTO is_muted_by_commenter;
        
        IF is_muted_by_commenter THEN
            RETURN false;
        END IF;
    END IF;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- MUTING HELPER FUNCTIONS
-- =====================================================

-- Function to get a user's muted users list
CREATE OR REPLACE FUNCTION get_user_muted_list(user_uuid UUID)
RETURNS TABLE(
    muted_user_id UUID,
    muted_username VARCHAR,
    mute_reason TEXT,
    muted_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        um.muted_id,
        u.username,
        um.reason,
        um.created_at
    FROM user_mutes um
    JOIN users u ON um.muted_id = u.id
    WHERE um.muter_id = user_uuid
    ORDER BY um.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to count muted users (for limits)
CREATE OR REPLACE FUNCTION count_user_mutes(user_uuid UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*) 
        FROM user_mutes 
        WHERE muter_id = user_uuid
    );
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE user_mutes IS 'User muting system matching AO3 implementation - muted users content is hidden';
COMMENT ON COLUMN user_mutes.muter_id IS 'User who is doing the muting';
COMMENT ON COLUMN user_mutes.muted_id IS 'User being muted';
COMMENT ON COLUMN user_mutes.reason IS 'Optional reason for muting';

COMMENT ON FUNCTION is_user_muted(UUID, UUID) IS 'Checks if one user has muted another user';
COMMENT ON FUNCTION get_user_muted_list(UUID) IS 'Returns list of users muted by the specified user';
COMMENT ON FUNCTION count_user_mutes(UUID) IS 'Returns count of users muted by the specified user';