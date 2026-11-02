-- OAuth provider: apps are vetted before they can be used (security audit,
-- phase 3).
--
-- Every app has a status:
--   pending    registered by a person, waiting for an admin
--   approved   an admin said yes; the only status that works
--   rejected   an admin said no
--   suspended  was working (or was waiting), an admin cut it off
--
-- is_active stays as a column because older code and tools read it, but it is
-- now only a copy of "status = 'approved'", kept true by the trigger below.
-- Nothing can make an app usable except moving it to approved.

ALTER TABLE oauth_clients
    ADD COLUMN status      TEXT NOT NULL DEFAULT 'approved',
    ADD COLUMN reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN reviewed_at TIMESTAMPTZ,
    ADD COLUMN review_note TEXT;

ALTER TABLE oauth_clients
    ADD CONSTRAINT oauth_clients_status_known
        CHECK (status IN ('pending', 'approved', 'rejected', 'suspended')),
    ADD CONSTRAINT oauth_clients_review_note_len
        CHECK (review_note IS NULL OR char_length(review_note) <= 1000);

-- Existing apps were made by admins, so they count as approved. One that was
-- already switched off stays unusable: it becomes suspended, not approved.
UPDATE oauth_clients SET status = 'suspended' WHERE NOT is_active;

CREATE OR REPLACE FUNCTION oauth_clients_sync_status() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- A row written with is_active = false and no status is a switched-off app.
        IF NEW.status = 'approved' AND NOT NEW.is_active THEN
            NEW.status := 'suspended';
        END IF;
    ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
        NULL; -- the status wins
    ELSIF NEW.is_active IS DISTINCT FROM OLD.is_active THEN
        IF NEW.is_active THEN
            -- Only an admin's approval may turn a waiting or refused app on.
            IF OLD.status IN ('pending', 'rejected') THEN
                RAISE EXCEPTION 'an app that is % can only be switched on by approving it', OLD.status;
            END IF;
            NEW.status := 'approved';
        ELSE
            NEW.status := 'suspended';
        END IF;
    END IF;
    NEW.is_active := (NEW.status = 'approved');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER oauth_clients_sync_status
    BEFORE INSERT OR UPDATE ON oauth_clients
    FOR EACH ROW EXECUTE FUNCTION oauth_clients_sync_status();

CREATE INDEX idx_oauth_clients_status ON oauth_clients (status, created_at);
CREATE INDEX idx_oauth_clients_owner_status ON oauth_clients (owner_id, status);

COMMENT ON COLUMN oauth_clients.status      IS 'pending, approved, rejected or suspended. Only approved apps work.';
COMMENT ON COLUMN oauth_clients.is_active   IS 'Always (status = approved); kept in step by a trigger. Read status instead.';
COMMENT ON COLUMN oauth_clients.reviewed_by IS 'The admin who last approved, rejected or suspended the app. NULL if done from the command line.';
COMMENT ON COLUMN oauth_clients.reviewed_at IS 'When the app was last reviewed.';
COMMENT ON COLUMN oauth_clients.review_note IS 'What the admin told the owner. Shown to the owner.';
