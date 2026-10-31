-- Background jobs (emails, notifications, exports) are rows in this table
-- instead of goroutines, so they survive restarts, are retried when they
-- fail, and can be seen and retried by hand: `nuclear jobs`.
CREATE TABLE jobs (
    id            BIGSERIAL PRIMARY KEY,
    kind          TEXT NOT NULL,                  -- which handler runs it, e.g. 'email'
    payload       JSONB NOT NULL DEFAULT '{}',
    run_at        TIMESTAMPTZ NOT NULL DEFAULT now(), -- not before this time
    attempts      INT NOT NULL DEFAULT 0,
    max_attempts  INT NOT NULL DEFAULT 10,       -- about four hours of retries (see backoff in internal/jobs)
    last_error    TEXT,
    locked_at     TIMESTAMPTZ,                    -- set while a worker runs it
    done_at       TIMESTAMPTZ,
    failed_at     TIMESTAMPTZ,                    -- gave up after max_attempts
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- What workers look for: jobs that are neither done nor failed.
CREATE INDEX idx_jobs_waiting ON jobs (run_at) WHERE done_at IS NULL AND failed_at IS NULL;
CREATE INDEX idx_jobs_failed ON jobs (failed_at) WHERE failed_at IS NOT NULL;
