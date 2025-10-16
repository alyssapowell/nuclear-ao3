-- =====================================================
-- ENHANCED NOTIFICATION SYSTEM
-- Extends the basic notification system with advanced features:
-- - User notification preferences and settings
-- - Content subscriptions
-- - Notification rules and smart filtering
-- - Batching and digest support
-- - Mention detection and parsing
-- =====================================================

-- Update notification types to include mentions and more granular events
ALTER TABLE notifications 
DROP CONSTRAINT IF EXISTS notification_type_check;

ALTER TABLE notifications 
ADD CONSTRAINT notification_type_check CHECK (type IN (
    'comment_reply', 'work_comment', 'work_kudos', 'work_bookmark',
    'user_follow', 'collection_invite', 'system_announcement',
    'work_update', 'series_update', 'tag_wrangling',
    'comment_mention', 'work_mention', 'comment_received',
    'comment_replied', 'kudos_received', 'bookmark_added',
    'gift_received', 'moderator_action', 'system_alert',
    'account_security', 'password_reset', 'new_work',
    'work_completed', 'series_updated'
));

-- Add new columns to support advanced notification features
ALTER TABLE notifications 
ADD COLUMN IF NOT EXISTS event VARCHAR(50),
ADD COLUMN IF NOT EXISTS priority VARCHAR(10) DEFAULT 'medium',
ADD COLUMN IF NOT EXISTS source_id UUID,
ADD COLUMN IF NOT EXISTS source_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS action_url TEXT,
ADD COLUMN IF NOT EXISTS actor_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS actor_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS extra_data JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS is_delivered BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS digest_id UUID,
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- Create indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_notifications_source ON notifications(source_id, source_type);
CREATE INDEX IF NOT EXISTS idx_notifications_actor ON notifications(actor_id);
CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications(priority);
CREATE INDEX IF NOT EXISTS idx_notifications_delivered ON notifications(is_delivered, delivered_at);
CREATE INDEX IF NOT EXISTS idx_notifications_digest ON notifications(digest_id);

-- =====================================================
-- USER NOTIFICATION PREFERENCES
-- =====================================================

CREATE TABLE notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    
    -- Global settings
    email_enabled BOOLEAN DEFAULT true,
    web_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT false,
    
    -- Event-specific preferences stored as JSONB for flexibility
    event_preferences JSONB DEFAULT '{}',
    
    -- Batching settings
    enable_batching BOOLEAN DEFAULT true,
    batch_frequency VARCHAR(20) DEFAULT 'daily',
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Anti-spam settings
    max_notifications_per_hour INTEGER DEFAULT 10,
    min_time_between_similar_minutes INTEGER DEFAULT 60,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- CONTENT SUBSCRIPTIONS
-- =====================================================

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    target_name VARCHAR(255),
    events TEXT[] DEFAULT '{}',
    frequency VARCHAR(20) DEFAULT 'immediate',
    last_notified TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    
    -- Advanced filtering options
    filter_completed BOOLEAN,
    filter_rating TEXT[],
    filter_warnings TEXT[],
    filter_tags TEXT[],
    min_word_count INTEGER,
    max_word_count INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT subscription_type_check CHECK (type IN (
        'work', 'series', 'author', 'tag', 'collection', 'user'
    )),
    CONSTRAINT subscription_frequency_check CHECK (frequency IN (
        'immediate', 'batched', 'daily', 'weekly', 'never'
    )),
    UNIQUE(user_id, type, target_id)
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id, is_active);
CREATE INDEX idx_subscriptions_target ON subscriptions(type, target_id, is_active);
CREATE INDEX idx_subscriptions_frequency ON subscriptions(frequency, last_notified);

-- =====================================================
-- NOTIFICATION RULES
-- =====================================================

CREATE TABLE notification_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Conditions (stored as JSONB for flexibility)
    events TEXT[] DEFAULT '{}',
    source_types TEXT[],
    actor_conditions JSONB DEFAULT '{}',
    content_filters JSONB DEFAULT '{}',
    time_conditions JSONB DEFAULT '{}',
    
    -- Actions
    action VARCHAR(20) NOT NULL DEFAULT 'allow',
    priority VARCHAR(10),
    force_channel VARCHAR(20),
    delay_minutes INTEGER,
    
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT rule_action_check CHECK (action IN (
        'allow', 'block', 'modify', 'batch', 'escalate'
    )),
    CONSTRAINT rule_priority_check CHECK (priority IS NULL OR priority IN (
        'high', 'medium', 'low'
    ))
);

CREATE INDEX idx_notification_rules_user ON notification_rules(user_id, is_active);

-- =====================================================
-- NOTIFICATION DIGESTS
-- =====================================================

CREATE TABLE notification_digests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    digest_type VARCHAR(20) NOT NULL,
    notifications JSONB DEFAULT '[]',
    sent_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT digest_type_check CHECK (digest_type IN ('daily', 'weekly')),
    CONSTRAINT digest_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX idx_notification_digests_user ON notification_digests(user_id, digest_type);
CREATE INDEX idx_notification_digests_status ON notification_digests(status, created_at);

-- Update existing notifications table to reference digests
ALTER TABLE notifications 
ADD CONSTRAINT fk_notifications_digest 
FOREIGN KEY (digest_id) REFERENCES notification_digests(id) ON DELETE SET NULL;

-- =====================================================
-- MENTION TRACKING
-- =====================================================

CREATE TABLE mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_type VARCHAR(20) NOT NULL, -- 'comment', 'work', 'note', etc.
    source_id UUID NOT NULL,
    mentioned_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mentioning_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    content_excerpt TEXT,
    position_start INTEGER,
    position_end INTEGER,
    notification_id UUID REFERENCES notifications(id) ON DELETE SET NULL,
    is_resolved BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT mention_source_type_check CHECK (source_type IN (
        'comment', 'work', 'series', 'bookmark', 'note'
    ))
);

CREATE INDEX idx_mentions_user ON mentions(mentioned_user_id, is_resolved);
CREATE INDEX idx_mentions_source ON mentions(source_type, source_id);
CREATE INDEX idx_mentions_notification ON mentions(notification_id);

-- =====================================================
-- NOTIFICATION DELIVERY TRACKING
-- =====================================================

CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempted_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT delivery_channel_check CHECK (channel IN (
        'email', 'push', 'sms', 'webhook', 'in_app'
    )),
    CONSTRAINT delivery_status_check CHECK (status IN (
        'pending', 'sent', 'delivered', 'failed', 'bounced'
    ))
);

CREATE INDEX idx_notification_deliveries_notification ON notification_deliveries(notification_id);
CREATE INDEX idx_notification_deliveries_status ON notification_deliveries(status, attempted_at);
CREATE INDEX idx_notification_deliveries_channel ON notification_deliveries(channel, status);

-- =====================================================
-- HELPER FUNCTIONS FOR MENTIONS
-- =====================================================

-- Function to extract mentions from text content
CREATE OR REPLACE FUNCTION extract_mentions(content TEXT)
RETURNS TABLE(username TEXT, position_start INTEGER, position_end INTEGER) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        TRIM(LEADING '@' FROM match[1]) as username,
        match_start::INTEGER as position_start,
        (match_start + LENGTH(match[0]) - 1)::INTEGER as position_end
    FROM (
        SELECT 
            regexp_matches(content, '@([a-zA-Z0-9_-]+)', 'gi') as match,
            strpos(content, regexp_matches(content, '@([a-zA-Z0-9_-]+)', 'gi')[1]) as match_start
    ) matches;
END;
$$ LANGUAGE plpgsql;

-- Function to create mention records
CREATE OR REPLACE FUNCTION create_mentions(
    p_source_type VARCHAR(20),
    p_source_id UUID,
    p_content TEXT,
    p_mentioning_user_id UUID DEFAULT NULL
) RETURNS INTEGER AS $$
DECLARE
    mention_record RECORD;
    mentioned_user_id UUID;
    created_count INTEGER := 0;
    notification_id UUID;
BEGIN
    -- Extract mentions from content
    FOR mention_record IN 
        SELECT * FROM extract_mentions(p_content)
    LOOP
        -- Find the mentioned user
        SELECT id INTO mentioned_user_id 
        FROM users 
        WHERE username = mention_record.username;
        
        IF mentioned_user_id IS NOT NULL THEN
            -- Create mention record
            INSERT INTO mentions (
                source_type, source_id, mentioned_user_id, mentioning_user_id,
                content_excerpt, position_start, position_end
            ) VALUES (
                p_source_type, p_source_id, mentioned_user_id, p_mentioning_user_id,
                SUBSTRING(p_content FROM mention_record.position_start FOR 100),
                mention_record.position_start, mention_record.position_end
            );
            
            -- Create notification for the mention
            SELECT create_notification(
                mentioned_user_id,
                CASE p_source_type 
                    WHEN 'comment' THEN 'comment_mention'
                    WHEN 'work' THEN 'work_mention'
                    ELSE 'user_mention'
                END,
                'You were mentioned',
                CASE p_source_type
                    WHEN 'comment' THEN 'You were mentioned in a comment'
                    WHEN 'work' THEN 'You were mentioned in a work'
                    ELSE 'You were mentioned'
                END,
                jsonb_build_object(
                    'source_type', p_source_type,
                    'source_id', p_source_id,
                    'mentioning_user_id', p_mentioning_user_id,
                    'excerpt', SUBSTRING(p_content FROM mention_record.position_start FOR 100)
                )
            ) INTO notification_id;
            
            -- Link the mention to the notification
            UPDATE mentions 
            SET notification_id = notification_id 
            WHERE mentioned_user_id = mentioned_user_id 
              AND source_id = p_source_id 
              AND source_type = p_source_type;
            
            created_count := created_count + 1;
        END IF;
    END LOOP;
    
    RETURN created_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- UPDATE COMMENT NOTIFICATION FUNCTIONS
-- =====================================================

-- Enhanced function to handle comment notifications with mentions
CREATE OR REPLACE FUNCTION handle_comment_notifications()
RETURNS TRIGGER AS $$
DECLARE
    work_author_id UUID;
    parent_comment_author_id UUID;
    notification_title TEXT;
    notification_message TEXT;
    work_title TEXT;
    commenter_name TEXT;
    mention_count INTEGER;
BEGIN
    -- Get work information
    SELECT w.user_id, w.title INTO work_author_id, work_title
    FROM works w 
    WHERE w.id = NEW.work_id;
    
    -- Get commenter name
    SELECT COALESCE(up.name, u.username, NEW.guest_name) INTO commenter_name
    FROM users u
    LEFT JOIN user_pseudonyms up ON NEW.pseudonym_id = up.id
    WHERE u.id = NEW.user_id;
    
    -- Handle mentions in comment content
    SELECT create_mentions(
        'comment',
        NEW.id,
        NEW.content,
        NEW.user_id
    ) INTO mention_count;
    
    -- Notify work author (if not commenting on their own work)
    IF work_author_id IS NOT NULL AND work_author_id != NEW.user_id THEN
        notification_title := format('New comment on "%s"', work_title);
        notification_message := format('%s commented on your work', COALESCE(commenter_name, 'Someone'));
        
        PERFORM create_notification(
            work_author_id,
            'work_comment',
            notification_title,
            notification_message,
            jsonb_build_object(
                'work_id', NEW.work_id,
                'comment_id', NEW.id,
                'commenter_id', NEW.user_id,
                'commenter_name', commenter_name,
                'work_title', work_title
            )
        );
    END IF;
    
    -- Handle reply notifications
    IF NEW.parent_comment_id IS NOT NULL THEN
        -- Get parent comment author
        SELECT c.user_id INTO parent_comment_author_id
        FROM comments c 
        WHERE c.id = NEW.parent_comment_id;
        
        -- Notify parent comment author (if not replying to themselves)
        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id != NEW.user_id THEN
            notification_title := 'Reply to your comment';
            notification_message := format('%s replied to your comment', COALESCE(commenter_name, 'Someone'));
            
            PERFORM create_notification(
                parent_comment_author_id,
                'comment_reply',
                notification_title,
                notification_message,
                jsonb_build_object(
                    'work_id', NEW.work_id,
                    'comment_id', NEW.id,
                    'parent_comment_id', NEW.parent_comment_id,
                    'commenter_id', NEW.user_id,
                    'commenter_name', commenter_name,
                    'work_title', work_title
                )
            );
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- DEFAULT NOTIFICATION PREFERENCES FOR NEW USERS
-- =====================================================

-- Function to create default notification preferences for a new user
CREATE OR REPLACE FUNCTION create_default_notification_preferences(p_user_id UUID)
RETURNS VOID AS $$
BEGIN
    INSERT INTO notification_preferences (
        user_id,
        email_enabled,
        web_enabled,
        push_enabled,
        event_preferences,
        enable_batching,
        batch_frequency,
        max_notifications_per_hour,
        min_time_between_similar_minutes
    ) VALUES (
        p_user_id,
        true,
        true,
        false,
        jsonb_build_object(
            'work_comment', jsonb_build_object(
                'enabled', true,
                'channels', '["email", "in_app"]',
                'frequency', 'immediate',
                'priority', 'high'
            ),
            'comment_reply', jsonb_build_object(
                'enabled', true,
                'channels', '["email", "in_app"]',
                'frequency', 'immediate',
                'priority', 'high'
            ),
            'comment_mention', jsonb_build_object(
                'enabled', true,
                'channels', '["email", "in_app"]',
                'frequency', 'immediate',
                'priority', 'high'
            ),
            'work_kudos', jsonb_build_object(
                'enabled', true,
                'channels', '["in_app"]',
                'frequency', 'daily',
                'priority', 'low'
            ),
            'work_update', jsonb_build_object(
                'enabled', true,
                'channels', '["email", "in_app"]',
                'frequency', 'immediate',
                'priority', 'medium'
            )
        ),
        true,
        'daily',
        10,
        60
    ) ON CONFLICT (user_id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Trigger to create default preferences for new users
CREATE OR REPLACE FUNCTION create_user_notification_defaults()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM create_default_notification_preferences(NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_user_notification_defaults_trigger
    AFTER INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION create_user_notification_defaults();

-- =====================================================
-- CLEANUP AND MAINTENANCE
-- =====================================================

-- Function to clean up old notifications
CREATE OR REPLACE FUNCTION cleanup_old_notifications(days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM notifications 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old
      AND is_read = true;
      
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old notification deliveries
CREATE OR REPLACE FUNCTION cleanup_old_deliveries(days_old INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM notification_deliveries 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old
      AND status IN ('delivered', 'failed');
      
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE notification_preferences IS 'User preferences for notification delivery and batching';
COMMENT ON TABLE subscriptions IS 'User subscriptions to content for automatic notifications';
COMMENT ON TABLE notification_rules IS 'User-defined rules for filtering and modifying notifications';
COMMENT ON TABLE notification_digests IS 'Batched notifications sent as digests';
COMMENT ON TABLE mentions IS 'Tracking of user mentions in content';
COMMENT ON TABLE notification_deliveries IS 'Tracking of notification delivery attempts across channels';

COMMENT ON FUNCTION extract_mentions IS 'Extracts @username mentions from text content';
COMMENT ON FUNCTION create_mentions IS 'Creates mention records and notifications for mentioned users';
COMMENT ON FUNCTION create_default_notification_preferences IS 'Creates default notification preferences for new users';
COMMENT ON FUNCTION cleanup_old_notifications IS 'Removes old read notifications to maintain database performance';