-- Nuclear AO3: Enhanced Comments and User Profiles Migration
-- This migration enhances the existing comment system and user profiles for AO3 parity
-- Adds threading, moderation, notifications, and comprehensive profile features

-- =====================================================
-- ENHANCED USER PROFILE FEATURES
-- =====================================================

-- User profile extensions for AO3-style profiles
ALTER TABLE users ADD COLUMN IF NOT EXISTS birth_date DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'UTC';
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_language VARCHAR(10) DEFAULT 'en';
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_visibility VARCHAR(20) DEFAULT 'public';
ALTER TABLE users ADD COLUMN IF NOT EXISTS work_visibility VARCHAR(20) DEFAULT 'public';
ALTER TABLE users ADD COLUMN IF NOT EXISTS comment_permissions VARCHAR(20) DEFAULT 'all';
ALTER TABLE users ADD COLUMN IF NOT EXISTS skin_theme VARCHAR(50) DEFAULT 'default';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_notifications JSONB DEFAULT '{
    "comment_replies": true,
    "kudos": true,
    "bookmarks": false,
    "work_updates": true,
    "system_announcements": true
}'::jsonb;

-- Add profile constraints
ALTER TABLE users ADD CONSTRAINT profile_visibility_check 
    CHECK (profile_visibility IN ('public', 'registered_users', 'friends', 'private'));
ALTER TABLE users ADD CONSTRAINT work_visibility_check 
    CHECK (work_visibility IN ('public', 'registered_users', 'private'));
ALTER TABLE users ADD CONSTRAINT comment_permissions_check 
    CHECK (comment_permissions IN ('all', 'registered_users', 'friends', 'none'));

-- User pseudonyms table (AO3 allows multiple pen names per user)
CREATE TABLE user_pseudonyms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name CITEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    description TEXT,
    icon_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, name)
);

CREATE INDEX idx_user_pseudonyms_user_id ON user_pseudonyms(user_id);
CREATE INDEX idx_user_pseudonyms_name ON user_pseudonyms(name);
CREATE INDEX idx_user_pseudonyms_default ON user_pseudonyms(user_id, is_default) WHERE is_default = true;

-- User friendships/relationships
CREATE TABLE user_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT user_relationships_status_check CHECK (status IN ('pending', 'accepted', 'blocked', 'rejected')),
    CONSTRAINT no_self_friendship CHECK (requester_id != addressee_id),
    UNIQUE(requester_id, addressee_id)
);

CREATE INDEX idx_user_relationships_requester ON user_relationships(requester_id, status);
CREATE INDEX idx_user_relationships_addressee ON user_relationships(addressee_id, status);

-- User blocking system
CREATE TABLE user_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT no_self_blocking CHECK (blocker_id != blocked_id),
    UNIQUE(blocker_id, blocked_id)
);

CREATE INDEX idx_user_blocks_blocker ON user_blocks(blocker_id);
CREATE INDEX idx_user_blocks_blocked ON user_blocks(blocked_id);

-- =====================================================
-- ENHANCED COMMENT SYSTEM
-- =====================================================

-- Add enhanced comment features
ALTER TABLE comments ADD COLUMN IF NOT EXISTS pseudonym_id UUID REFERENCES user_pseudonyms(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS guest_name VARCHAR(100);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS guest_email CITEXT;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS edited_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS thread_level INTEGER DEFAULT 0;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS is_spam BOOLEAN DEFAULT false;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS kudos_count INTEGER DEFAULT 0;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS reply_count INTEGER DEFAULT 0;

-- Comment threading constraints
ALTER TABLE comments ADD CONSTRAINT comment_identity_check CHECK (
    (user_id IS NOT NULL AND pseudonym_id IS NOT NULL AND guest_name IS NULL) OR
    (user_id IS NULL AND pseudonym_id IS NULL AND guest_name IS NOT NULL)
);

-- Create indexes for enhanced comments
CREATE INDEX idx_comments_pseudonym ON comments(pseudonym_id) WHERE pseudonym_id IS NOT NULL;
CREATE INDEX idx_comments_thread_level ON comments(work_id, thread_level, created_at) WHERE work_id IS NOT NULL;
CREATE INDEX idx_comments_guest ON comments(guest_name, guest_email) WHERE guest_name IS NOT NULL;
CREATE INDEX idx_comments_ip ON comments(ip_address);
CREATE INDEX idx_comments_spam ON comments(is_spam) WHERE is_spam = true;

-- Comment kudos table (users can kudos individual comments)
CREATE TABLE comment_kudos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    pseudonym_id UUID REFERENCES user_pseudonyms(id) ON DELETE SET NULL,
    guest_session VARCHAR(255),
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT comment_kudos_identity CHECK (
        (user_id IS NOT NULL AND pseudonym_id IS NOT NULL AND guest_session IS NULL) OR
        (user_id IS NULL AND pseudonym_id IS NULL AND guest_session IS NOT NULL)
    )
);

CREATE INDEX idx_comment_kudos_comment ON comment_kudos(comment_id);
CREATE INDEX idx_comment_kudos_user ON comment_kudos(user_id, pseudonym_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX idx_comment_kudos_unique_user ON comment_kudos(comment_id, user_id, pseudonym_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX idx_comment_kudos_unique_guest ON comment_kudos(comment_id, guest_session) WHERE guest_session IS NOT NULL;

-- Comment reports/flags
CREATE TABLE comment_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    reporter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT report_reason_check CHECK (reason IN ('spam', 'harassment', 'off_topic', 'inappropriate', 'copyright', 'other')),
    CONSTRAINT report_status_check CHECK (status IN ('pending', 'reviewed', 'resolved', 'dismissed'))
);

CREATE INDEX idx_comment_reports_comment ON comment_reports(comment_id);
CREATE INDEX idx_comment_reports_reporter ON comment_reports(reporter_id);
CREATE INDEX idx_comment_reports_status ON comment_reports(status);

-- =====================================================
-- NOTIFICATIONS SYSTEM
-- =====================================================

-- User notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB, -- Additional notification data
    is_read BOOLEAN DEFAULT false,
    is_emailed BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT notification_type_check CHECK (type IN (
        'comment_reply', 'work_comment', 'work_kudos', 'work_bookmark',
        'user_follow', 'collection_invite', 'system_announcement',
        'work_update', 'series_update', 'tag_wrangling'
    ))
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX idx_notifications_type ON notifications(type);

-- =====================================================
-- USER STATISTICS AND ACTIVITY
-- =====================================================

-- User statistics table
CREATE TABLE user_statistics (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    works_count INTEGER DEFAULT 0,
    series_count INTEGER DEFAULT 0,
    bookmarks_count INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    kudos_given_count INTEGER DEFAULT 0,
    kudos_received_count INTEGER DEFAULT 0,
    words_written INTEGER DEFAULT 0,
    last_work_date TIMESTAMP WITH TIME ZONE,
    join_date TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User activity log for tracking actions
CREATE TABLE user_activity_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50), -- 'work', 'comment', 'bookmark', etc.
    entity_id UUID,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_user_activity_log_user_id ON user_activity_log(user_id, created_at DESC);
CREATE INDEX idx_user_activity_log_action ON user_activity_log(action);
CREATE INDEX idx_user_activity_log_entity ON user_activity_log(entity_type, entity_id);

-- =====================================================
-- TRIGGERS AND FUNCTIONS FOR ENHANCED FEATURES
-- =====================================================

-- Function to update comment statistics
CREATE OR REPLACE FUNCTION update_comment_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'comment_kudos' THEN
        -- Update comment kudos count
        UPDATE comments SET 
            kudos_count = (SELECT COUNT(*) FROM comment_kudos WHERE comment_id = NEW.comment_id)
        WHERE id = NEW.comment_id;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        -- Update parent comment reply count
        IF NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET 
                reply_count = (SELECT COUNT(*) FROM comments WHERE parent_comment_id = NEW.parent_comment_id AND is_deleted = false)
            WHERE id = NEW.parent_comment_id;
        END IF;
        
        -- Update comment thread level
        IF NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET 
                thread_level = (SELECT thread_level + 1 FROM comments WHERE id = NEW.parent_comment_id)
            WHERE id = NEW.id;
        END IF;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Triggers for comment statistics
CREATE TRIGGER update_comment_stats_kudos
    AFTER INSERT OR DELETE ON comment_kudos
    FOR EACH ROW EXECUTE FUNCTION update_comment_stats();

CREATE TRIGGER update_comment_stats_replies
    AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_comment_stats();

-- Function to update user statistics
CREATE OR REPLACE FUNCTION update_user_stats()
RETURNS TRIGGER AS $$
DECLARE
    target_user_id UUID;
BEGIN
    -- Determine which user's stats to update
    IF TG_TABLE_NAME = 'works' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'comments' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'bookmarks' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    ELSIF TG_TABLE_NAME = 'kudos' THEN
        target_user_id := COALESCE(NEW.user_id, OLD.user_id);
    END IF;
    
    -- Update user statistics
    INSERT INTO user_statistics (user_id, join_date) 
    VALUES (target_user_id, (SELECT created_at FROM users WHERE id = target_user_id))
    ON CONFLICT (user_id) DO UPDATE SET
        works_count = (
            SELECT COUNT(*) FROM works 
            WHERE user_id = target_user_id AND is_draft = false
        ),
        series_count = (
            SELECT COUNT(*) FROM series WHERE user_id = target_user_id
        ),
        bookmarks_count = (
            SELECT COUNT(*) FROM bookmarks WHERE user_id = target_user_id
        ),
        comments_count = (
            SELECT COUNT(*) FROM comments 
            WHERE user_id = target_user_id AND is_deleted = false
        ),
        kudos_given_count = (
            SELECT COUNT(*) FROM kudos WHERE user_id = target_user_id
        ),
        kudos_received_count = (
            SELECT COUNT(*) FROM kudos k 
            JOIN works w ON k.work_id = w.id 
            WHERE w.user_id = target_user_id
        ),
        words_written = (
            SELECT COALESCE(SUM(word_count), 0) FROM works 
            WHERE user_id = target_user_id AND is_draft = false
        ),
        last_work_date = (
            SELECT MAX(published_at) FROM works 
            WHERE user_id = target_user_id AND is_draft = false
        ),
        updated_at = NOW();
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Triggers for user statistics
CREATE TRIGGER update_user_stats_works
    AFTER INSERT OR UPDATE OR DELETE ON works
    FOR EACH ROW EXECUTE FUNCTION update_user_stats();

CREATE TRIGGER update_user_stats_comments
    AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_user_stats();

CREATE TRIGGER update_user_stats_bookmarks
    AFTER INSERT OR DELETE ON bookmarks
    FOR EACH ROW EXECUTE FUNCTION update_user_stats();

CREATE TRIGGER update_user_stats_kudos
    AFTER INSERT OR DELETE ON kudos
    FOR EACH ROW EXECUTE FUNCTION update_user_stats();

-- Function to create notification
CREATE OR REPLACE FUNCTION create_notification(
    p_user_id UUID,
    p_type VARCHAR(50),
    p_title VARCHAR(255),
    p_message TEXT DEFAULT NULL,
    p_data JSONB DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    notification_id UUID;
BEGIN
    INSERT INTO notifications (user_id, type, title, message, data)
    VALUES (p_user_id, p_type, p_title, p_message, p_data)
    RETURNING id INTO notification_id;
    
    RETURN notification_id;
END;
$$ language 'plpgsql';

-- Function to handle comment notifications
CREATE OR REPLACE FUNCTION handle_comment_notifications()
RETURNS TRIGGER AS $$
DECLARE
    work_author_id UUID;
    parent_comment_author_id UUID;
    notification_title TEXT;
    notification_message TEXT;
BEGIN
    -- Get work author for work comments
    IF NEW.work_id IS NOT NULL THEN
        SELECT user_id INTO work_author_id FROM works WHERE id = NEW.work_id;
        
        -- Notify work author of new comment (if not commenting on own work)
        IF work_author_id IS NOT NULL AND work_author_id != NEW.user_id THEN
            notification_title := 'New comment on your work';
            notification_message := 'Someone commented on your work';
            
            PERFORM create_notification(
                work_author_id,
                'work_comment',
                notification_title,
                notification_message,
                jsonb_build_object('comment_id', NEW.id, 'work_id', NEW.work_id)
            );
        END IF;
    END IF;
    
    -- Get parent comment author for comment replies
    IF NEW.parent_comment_id IS NOT NULL THEN
        SELECT user_id INTO parent_comment_author_id FROM comments WHERE id = NEW.parent_comment_id;
        
        -- Notify parent comment author of reply (if not replying to own comment)
        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id != NEW.user_id THEN
            notification_title := 'Reply to your comment';
            notification_message := 'Someone replied to your comment';
            
            PERFORM create_notification(
                parent_comment_author_id,
                'comment_reply',
                notification_title,
                notification_message,
                jsonb_build_object('comment_id', NEW.id, 'parent_comment_id', NEW.parent_comment_id)
            );
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for comment notifications
CREATE TRIGGER handle_comment_notifications_trigger
    AFTER INSERT ON comments
    FOR EACH ROW EXECUTE FUNCTION handle_comment_notifications();

-- =====================================================
-- VIEWS FOR ENHANCED FEATURES
-- =====================================================

-- User profile view with statistics
CREATE VIEW user_profiles AS
SELECT 
    u.*,
    us.works_count,
    us.series_count,
    us.bookmarks_count,
    us.comments_count,
    us.kudos_given_count,
    us.kudos_received_count,
    us.words_written,
    us.last_work_date,
    ARRAY_AGG(DISTINCT up.name ORDER BY up.is_default DESC, up.name) FILTER (WHERE up.name IS NOT NULL) as pseudonyms,
    COUNT(DISTINCT ur1.addressee_id) FILTER (WHERE ur1.status = 'accepted') as friends_count,
    COUNT(DISTINCT ur2.requester_id) FILTER (WHERE ur2.status = 'accepted') as friends_count_alt
FROM users u
LEFT JOIN user_statistics us ON u.id = us.user_id
LEFT JOIN user_pseudonyms up ON u.id = up.user_id
LEFT JOIN user_relationships ur1 ON u.id = ur1.requester_id
LEFT JOIN user_relationships ur2 ON u.id = ur2.addressee_id
GROUP BY u.id, us.works_count, us.series_count, us.bookmarks_count, us.comments_count, 
         us.kudos_given_count, us.kudos_received_count, us.words_written, us.last_work_date;

-- Enhanced comments view with threading
CREATE VIEW comments_with_details AS
SELECT 
    c.*,
    COALESCE(up.name, u.username, c.guest_name) as author_name,
    u.id as author_user_id,
    up.id as author_pseudonym_id,
    CASE 
        WHEN c.guest_name IS NOT NULL THEN 'guest'
        WHEN u.id IS NOT NULL THEN 'user'
        ELSE 'unknown'
    END as author_type,
    w.title as work_title,
    w.user_id as work_author_id,
    parent_c.content as parent_content,
    COALESCE(parent_up.name, parent_u.username) as parent_author_name
FROM comments c
LEFT JOIN users u ON c.user_id = u.id
LEFT JOIN user_pseudonyms up ON c.pseudonym_id = up.id
LEFT JOIN works w ON c.work_id = w.id
LEFT JOIN comments parent_c ON c.parent_comment_id = parent_c.id
LEFT JOIN users parent_u ON parent_c.user_id = parent_u.id
LEFT JOIN user_pseudonyms parent_up ON parent_c.pseudonym_id = parent_up.id
WHERE c.is_deleted = false;

-- User dashboard view
CREATE VIEW user_dashboard AS
SELECT 
    u.id,
    u.username,
    u.display_name,
    COUNT(DISTINCT n.id) FILTER (WHERE n.is_read = false) as unread_notifications,
    COUNT(DISTINCT w.id) FILTER (WHERE w.is_draft = false) as published_works,
    COUNT(DISTINCT w2.id) FILTER (WHERE w2.is_draft = true) as draft_works,
    COUNT(DISTINCT b.id) as bookmarks,
    COUNT(DISTINCT s.id) as series,
    COALESCE(SUM(w.hit_count), 0) as total_hits,
    COALESCE(SUM(w.kudos_count), 0) as total_kudos,
    COALESCE(SUM(w.comment_count), 0) as total_comments,
    MAX(w.published_at) as last_published
FROM users u
LEFT JOIN notifications n ON u.id = n.user_id
LEFT JOIN works w ON u.id = w.user_id AND w.is_draft = false
LEFT JOIN works w2 ON u.id = w2.user_id AND w2.is_draft = true
LEFT JOIN bookmarks b ON u.id = b.user_id
LEFT JOIN series s ON u.id = s.user_id
GROUP BY u.id, u.username, u.display_name;

-- =====================================================
-- SAMPLE DATA FOR TESTING
-- =====================================================

-- Insert sample pseudonyms
INSERT INTO user_pseudonyms (user_id, name, is_default, description) 
SELECT 
    id, 
    username, 
    true, 
    'Default pseudonym'
FROM users 
WHERE id IN (
    '123e4567-e89b-12d3-a456-426614174000',
    '123e4567-e89b-12d3-a456-426614174001'
)
ON CONFLICT (user_id, name) DO NOTHING;

-- Initialize user statistics for existing users
INSERT INTO user_statistics (user_id, join_date)
SELECT id, created_at FROM users
ON CONFLICT (user_id) DO NOTHING;

COMMENT ON TABLE user_pseudonyms IS 'Multiple pen names per user, AO3-style pseudonym system';
COMMENT ON TABLE user_relationships IS 'Friend connections and social relationships between users';
COMMENT ON TABLE comment_kudos IS 'Kudos/likes system for individual comments';
COMMENT ON TABLE notifications IS 'User notification system for all platform events';
COMMENT ON TABLE user_statistics IS 'Aggregated user activity and content statistics';
COMMENT ON FUNCTION create_notification IS 'Helper function to create notifications with proper formatting';
COMMENT ON VIEW user_profiles IS 'Complete user profiles with statistics and social connections';
COMMENT ON VIEW comments_with_details IS 'Enhanced comment view with full threading and author information';