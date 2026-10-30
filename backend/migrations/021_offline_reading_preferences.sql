-- Add offline reading preferences to users table
ALTER TABLE users ADD COLUMN default_offline_reading VARCHAR(20) DEFAULT 'pwa_only' CHECK (default_offline_reading IN ('files_and_pwa', 'pwa_only', 'none'));

-- Add per-work offline reading overrides to works table  
ALTER TABLE works ADD COLUMN offline_reading_override VARCHAR(20) CHECK (offline_reading_override IN ('files_and_pwa', 'pwa_only', 'none', 'use_default'));
ALTER TABLE works ADD COLUMN offline_reading_override_reason TEXT;

-- Set default values
UPDATE works SET offline_reading_override = 'use_default' WHERE offline_reading_override IS NULL;

-- Index for quick lookups
CREATE INDEX idx_works_offline_reading ON works(offline_reading_override);
CREATE INDEX idx_users_default_offline_reading ON users(default_offline_reading);

-- Add to audit log for tracking changes
CREATE TABLE offline_reading_preference_changes (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    work_id UUID, -- NULL for profile default changes
    old_setting VARCHAR(20),
    new_setting VARCHAR(20),
    reason TEXT,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (work_id) REFERENCES works(id)
);