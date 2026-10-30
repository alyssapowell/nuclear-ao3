-- can_user_view_work checked works.restricted_to_users, but the app stores
-- "logged-in users only" in works.restricted and never writes
-- restricted_to_users. So works restricted to logged-in users were readable
-- by anyone logged out. Check both columns.

CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_blocked BOOLEAN := false;
    is_muted BOOLEAN := false;
    is_author BOOLEAN := false;
BEGIN
    -- Get work privacy settings
    SELECT restricted, restricted_to_users, restricted_to_adults, status, user_id, is_anonymous, in_anon_collection
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
    IF (work_record.restricted OR work_record.restricted_to_users) AND viewer_uuid IS NULL THEN
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
