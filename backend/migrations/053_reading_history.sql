-- Reading history and Marked for Later (AO3: the readings table and the
-- history_enabled preference).
--
-- One row per (account, work). Readings belong to the account and are only
-- ever shown to it: there is no pseud on them and no public or app route
-- reads them. They go with the account (or the work) when either is deleted.
--
-- users.history_enabled is the preference. On by default, as in AO3.

ALTER TABLE users ADD COLUMN IF NOT EXISTS history_enabled BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS readings (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_id         UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    last_viewed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    view_count      INTEGER NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    last_chapter    INTEGER,
    marked_for_later BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, work_id)
);

-- The history page: newest viewed first, and the to-read list.
CREATE INDEX IF NOT EXISTS idx_readings_user_viewed ON readings (user_id, last_viewed_at DESC);
CREATE INDEX IF NOT EXISTS idx_readings_user_marked ON readings (user_id, last_viewed_at DESC) WHERE marked_for_later;
