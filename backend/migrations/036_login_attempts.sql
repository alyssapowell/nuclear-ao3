-- Password-guessing protection for login, shared by every copy of the app.
--
-- failed_login_attempts counts wrong passwords since the account last had a
-- good login (or a password reset, or a lock that ran out). When it reaches
-- the limit, locked_until is set and logins for the account are refused until
-- then. Both are kept in the database, not in the app's memory, so the count
-- holds across several app copies and across restarts.
ALTER TABLE users
    ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN locked_until TIMESTAMPTZ;
