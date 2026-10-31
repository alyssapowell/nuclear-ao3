-- The record of everything an administrator or moderator did: hiding and
-- unhiding, spam marks, suspensions and bans, role changes. One row per
-- action, written in the same transaction as the change itself.
--
-- AO3 keeps this in admin_activities (hide, unhide, spam, destroy); this
-- table also covers suspensions and role changes, which AO3 keeps in
-- log_items and in the roles tables.
--
-- The table is append-only. The triggers below refuse UPDATE, DELETE and
-- TRUNCATE for every role, so a bug or a stolen application login cannot
-- quietly rewrite history. They do not stop a database superuser (or the
-- table owner), who can drop the triggers or the table. Backups are the real
-- record; this table is what you read day to day.
--
-- actor_user_id has no foreign key on purpose: a foreign key to users would
-- UPDATE this table (set it to NULL) when an account is deleted, which the
-- triggers refuse. actor_label keeps the username as it was at the time.

CREATE TABLE admin_actions (
    id            BIGSERIAL PRIMARY KEY,
    at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_user_id UUID,
    actor_label   TEXT NOT NULL,      -- the username, or 'cli:<operating system user>'
    action        TEXT NOT NULL,      -- e.g. 'hide', 'unhide', 'spam', 'suspend', 'ban', 'grant_role'
    target_type   TEXT NOT NULL,      -- 'work', 'chapter', 'series', 'bookmark', 'comment', 'user'
    target_id     TEXT NOT NULL,
    reason        TEXT,
    details       JSONB
);

CREATE INDEX idx_admin_actions_at ON admin_actions (at DESC);
CREATE INDEX idx_admin_actions_target ON admin_actions (target_type, target_id);
CREATE INDEX idx_admin_actions_actor ON admin_actions (actor_label);

CREATE OR REPLACE FUNCTION admin_actions_are_append_only()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'admin_actions is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'integrity_constraint_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER admin_actions_no_update_delete
    BEFORE UPDATE OR DELETE ON admin_actions
    FOR EACH ROW EXECUTE FUNCTION admin_actions_are_append_only();

CREATE TRIGGER admin_actions_no_truncate
    BEFORE TRUNCATE ON admin_actions
    FOR EACH STATEMENT EXECUTE FUNCTION admin_actions_are_append_only();

REVOKE UPDATE, DELETE, TRUNCATE ON admin_actions FROM PUBLIC;
