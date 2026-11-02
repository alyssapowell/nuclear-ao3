-- OAuth provider: apps can't link people, and see only a chosen pseud
-- (security audit, phase 2).
--
-- 1. oauth_subjects: every app sees a different, stable, random "sub" for the
--    same person, so two apps can't compare notes to find the same user.
-- 2. user_consents.pseud_id: the one pseud the person chose to share with the
--    app when they approved it. If that pseud is deleted the column goes
--    NULL, and the app has to ask again; it never falls back to another
--    pseud, which could show that two pseuds are one person.
-- 3. user_consents.share_email: set only when the person explicitly approved
--    the email scope.

CREATE TABLE oauth_subjects (
    client_id UUID NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- 32 random bytes, base64url (43 characters). Not derived from the
    -- account id, so nothing about it can be worked out or compared.
    sub       TEXT NOT NULL CHECK (sub ~ '^[A-Za-z0-9_-]{43}$')
);

CREATE UNIQUE INDEX idx_oauth_subjects_client_user ON oauth_subjects (client_id, user_id);
CREATE UNIQUE INDEX idx_oauth_subjects_client_sub  ON oauth_subjects (client_id, sub);
-- Deleting a person goes through user_id.
CREATE INDEX idx_oauth_subjects_user ON oauth_subjects (user_id);

COMMENT ON TABLE oauth_subjects IS 'The per-app (pairwise) subject an app sees for a person. Created on first use, deleted with the app or the person.';

-- Existing approvals never chose a pseud or explicitly approved email in
-- this form, so they start with neither. An app that wants the profile or
-- the email has to ask again.
ALTER TABLE user_consents
    ADD COLUMN pseud_id    UUID REFERENCES pseuds(id) ON DELETE SET NULL,
    ADD COLUMN share_email BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN user_consents.pseud_id    IS 'The pseud the person chose to share with the app (scope profile). NULL when none was chosen or it was deleted.';
COMMENT ON COLUMN user_consents.share_email IS 'True only if the person explicitly approved the email scope for this app.';

-- Keep the scope descriptions the consent screen shows true to what apps get.
UPDATE oauth_scopes SET description = 'See the one pseud (name and id) you choose to share with this app' WHERE name = 'profile';
UPDATE oauth_scopes SET description = 'See the email address of your account' WHERE name = 'email';
