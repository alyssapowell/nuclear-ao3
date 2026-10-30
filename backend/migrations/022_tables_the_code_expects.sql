-- Tables the Go code reads and writes that no earlier migration created.
-- Before this, notifications, subscriptions, notification preferences,
-- profile preferences, collection membership and hit statistics all failed
-- with "relation does not exist" on a fresh database.
--
-- Column lists match the queries in:
--   backend/internal/notify/repositories.go
--   backend/internal/auth/user_profile_handlers.go
--   backend/internal/works/handlers.go

-- In-app notifications (notify package)
CREATE TABLE IF NOT EXISTS notification_items (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    source_id UUID NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    action_url TEXT,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_name VARCHAR(255),
    extra_data JSONB,
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_delivered BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE,
    delivered_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_notification_items_user ON notification_items(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_items_unread ON notification_items(user_id) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notification_items_undelivered ON notification_items(user_id, created_at) WHERE is_delivered = false;

-- Subscriptions to works, series and authors (notify package)
CREATE TABLE IF NOT EXISTS content_subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    filter_completed BOOLEAN,
    filter_rating JSONB,
    filter_tags JSONB,
    min_word_count INTEGER,
    max_word_count INTEGER,
    UNIQUE (user_id, target_type, target_id)
);
CREATE INDEX IF NOT EXISTS idx_content_subscriptions_target ON content_subscriptions(target_type, target_id) WHERE is_active;

-- Notification settings (notify package)
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_enabled BOOLEAN NOT NULL DEFAULT true,
    web_enabled BOOLEAN NOT NULL DEFAULT true,
    push_enabled BOOLEAN NOT NULL DEFAULT false,
    quiet_hours_start TIMESTAMP WITH TIME ZONE,
    quiet_hours_end TIMESTAMP WITH TIME ZONE,
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    event_preferences JSONB NOT NULL DEFAULT '{}',
    enable_batching BOOLEAN NOT NULL DEFAULT false,
    batch_frequency VARCHAR(20) NOT NULL DEFAULT 'immediate',
    max_notifications_per_hour INTEGER NOT NULL DEFAULT 50,
    min_time_between_similar BIGINT NOT NULL DEFAULT 0, -- nanoseconds (Go time.Duration)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Profile privacy and skin choice (auth package)
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    profile_visibility VARCHAR(20) DEFAULT 'public',
    work_visibility VARCHAR(20) DEFAULT 'public',
    comment_permissions VARCHAR(20) DEFAULT 'all',
    skin_theme VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Works in collections (works package). Replaces collection_works from 002,
-- which no code used.
CREATE TABLE IF NOT EXISTS collection_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_approved BOOLEAN NOT NULL DEFAULT false,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE,
    UNIQUE (collection_id, work_id)
);
CREATE INDEX IF NOT EXISTS idx_collection_items_work ON collection_items(work_id);
DROP TABLE IF EXISTS collection_works;

-- Per-day hit history for work statistics (works package)
CREATE TABLE IF NOT EXISTS work_hits (
    id BIGSERIAL PRIMARY KEY,
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    hit_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_work_hits_work_date ON work_hits(work_id, hit_date);
