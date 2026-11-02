-- OAuth provider: an admin's decision applies to the version they looked at,
-- and an owner can't rename an approved app without review.
--
-- revision goes up by one on every change to an app's row (an owner's edit,
-- an admin's decision, a secret reset). The review queue shows the revision,
-- and a decision must send it back; if the app changed in between, the
-- decision is refused and the admin looks again. A trigger does the
-- counting, so no code path can forget it.
--
-- proposed_* hold display changes (name, website, logo, description) that an
-- owner made to an approved app. People keep seeing the approved values until
-- an admin approves the change.

ALTER TABLE oauth_clients
    ADD COLUMN revision               INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN proposed_name          TEXT,
    ADD COLUMN proposed_description   TEXT,
    ADD COLUMN proposed_website       TEXT,
    ADD COLUMN proposed_logo_url      TEXT,
    ADD COLUMN display_review_pending BOOLEAN NOT NULL DEFAULT false;

CREATE OR REPLACE FUNCTION oauth_clients_bump_revision() RETURNS trigger AS $$
BEGIN
    NEW.revision := OLD.revision + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER oauth_clients_bump_revision
    BEFORE UPDATE ON oauth_clients
    FOR EACH ROW EXECUTE FUNCTION oauth_clients_bump_revision();

CREATE INDEX idx_oauth_clients_display_pending ON oauth_clients (created_at) WHERE display_review_pending;
