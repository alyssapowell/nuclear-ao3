-- Short-lived OAuth state (sign-in requests waiting for login, consent
-- screens waiting for an answer). Rows expire after about ten minutes and are
-- removed by the jobs cleanup.
CREATE TABLE oauth_pending (
    key        TEXT PRIMARY KEY,           -- e.g. 'consent:<id>' or 'auth_req:<id>'
    value      JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_pending_expires ON oauth_pending (expires_at);
