-- OAuth provider hardening (security audit, phases 0 and 1).
--
-- 1. Tokens and codes are stored only as a SHA-256 hash, so a copy of the
--    database cannot be used to act as an app or a user.
-- 2. Refresh tokens belong to a "family" (every token that came from one
--    sign-in), so reusing an old one can cancel the whole family.
-- 3. Codes and tokens are tied together, so a code that is used twice
--    cancels what the first use produced.
-- 4. The client-credentials grant, plain PKCE, admin-only scopes and long
--    lifetimes are gone from stored data, and the table refuses them.
--
-- Nothing has been published, so every existing token, code and pending
-- request is deleted rather than converted. Anyone who is signed in to an
-- app through this provider has to sign in again.

DELETE FROM oauth_refresh_tokens;
DELETE FROM oauth_access_tokens;
DELETE FROM authorization_codes;
DELETE FROM oauth_pending;

-- ---------------------------------------------------------------------
-- Hashes at rest
-- ---------------------------------------------------------------------

ALTER TABLE oauth_access_tokens  RENAME COLUMN token TO token_hash;
ALTER TABLE oauth_refresh_tokens RENAME COLUMN token TO token_hash;
ALTER TABLE authorization_codes  RENAME COLUMN code  TO code_hash;

-- The UNIQUE constraints already index these columns.
DROP INDEX idx_oauth_access_tokens_token;
DROP INDEX idx_oauth_refresh_tokens_token;

-- A raw token is URL-safe base64 and can never look like 64 hex digits, so
-- this stops a plain value from being stored by mistake.
ALTER TABLE oauth_access_tokens  ADD CONSTRAINT oauth_access_tokens_hash_is_sha256  CHECK (token_hash ~ '^[0-9a-f]{64}$');
ALTER TABLE oauth_refresh_tokens ADD CONSTRAINT oauth_refresh_tokens_hash_is_sha256 CHECK (token_hash ~ '^[0-9a-f]{64}$');
ALTER TABLE authorization_codes  ADD CONSTRAINT authorization_codes_hash_is_sha256  CHECK (code_hash ~ '^[0-9a-f]{64}$');

COMMENT ON COLUMN oauth_access_tokens.token_hash  IS 'SHA-256 (hex) of the access token. The token itself is never stored.';
COMMENT ON COLUMN oauth_refresh_tokens.token_hash IS 'SHA-256 (hex) of the refresh token. The token itself is never stored.';
COMMENT ON COLUMN authorization_codes.code_hash   IS 'SHA-256 (hex) of the authorization code. The code itself is never stored.';

-- ---------------------------------------------------------------------
-- Families, and the code each token came from
-- ---------------------------------------------------------------------

-- The tables are empty here, so the new columns can be NOT NULL.
ALTER TABLE oauth_access_tokens
    ADD COLUMN family_id UUID NOT NULL,
    ADD COLUMN code_hash TEXT;
ALTER TABLE oauth_refresh_tokens
    ADD COLUMN family_id         UUID NOT NULL,
    -- When the first token of the family was issued. A family can be
    -- refreshed for at most 90 days after this, however often it is used.
    ADD COLUMN family_created_at TIMESTAMPTZ NOT NULL,
    ADD COLUMN code_hash         TEXT;

CREATE INDEX idx_oauth_access_tokens_family  ON oauth_access_tokens (family_id);
CREATE INDEX idx_oauth_access_tokens_code    ON oauth_access_tokens (code_hash) WHERE code_hash IS NOT NULL;
CREATE INDEX idx_oauth_refresh_tokens_family ON oauth_refresh_tokens (family_id);
CREATE INDEX idx_oauth_refresh_tokens_code   ON oauth_refresh_tokens (code_hash) WHERE code_hash IS NOT NULL;

-- Every token belongs to a person again: the client-credentials grant (the
-- only thing that made user_id optional, migration 043) is removed.
ALTER TABLE oauth_access_tokens ALTER COLUMN user_id SET NOT NULL;

-- ---------------------------------------------------------------------
-- PKCE with S256 is required for every code
-- ---------------------------------------------------------------------

ALTER TABLE authorization_codes
    ALTER COLUMN code_challenge SET NOT NULL,
    ALTER COLUMN code_challenge_method SET NOT NULL,
    ADD CONSTRAINT authorization_codes_pkce_s256
        CHECK (code_challenge_method = 'S256' AND code_challenge ~ '^[A-Za-z0-9_-]{43}$');

COMMENT ON COLUMN authorization_codes.code_challenge        IS 'PKCE code challenge (S256, 43 characters). Required for every client.';
COMMENT ON COLUMN authorization_codes.code_challenge_method IS 'Always S256. "plain" is not supported.';

-- ---------------------------------------------------------------------
-- Clients: no client credentials, no admin scopes, bounded lifetimes
-- ---------------------------------------------------------------------

-- An app that existed only for the client-credentials grant has nothing
-- left it can do, so switch it off instead of quietly turning it into
-- something else.
UPDATE oauth_clients SET is_active = false
 WHERE 'client_credentials' = ANY(grant_types) AND NOT ('authorization_code' = ANY(grant_types));

UPDATE oauth_clients SET grant_types = ARRAY(
    SELECT g FROM unnest(grant_types) g WHERE g IN ('authorization_code', 'refresh_token'));
UPDATE oauth_clients SET grant_types = ARRAY['authorization_code'] WHERE cardinality(grant_types) = 0;

UPDATE oauth_clients SET response_types = ARRAY(SELECT r FROM unnest(response_types) r WHERE r = 'code');
UPDATE oauth_clients SET response_types = ARRAY['code'] WHERE cardinality(response_types) = 0;

UPDATE oauth_clients SET scopes = ARRAY(
    SELECT s FROM unnest(scopes) s WHERE s NOT IN (SELECT name FROM oauth_scopes WHERE is_admin_only));
UPDATE user_consents SET scopes = ARRAY(
    SELECT s FROM unnest(scopes) s WHERE s NOT IN (SELECT name FROM oauth_scopes WHERE is_admin_only));

UPDATE oauth_clients SET
    access_token_ttl  = LEAST(GREATEST(access_token_ttl, 300), 3600),
    refresh_token_ttl = LEAST(GREATEST(refresh_token_ttl, 3600), 2592000);

ALTER TABLE oauth_clients
    ADD CONSTRAINT oauth_clients_grant_types_allowed
        CHECK (cardinality(grant_types) > 0 AND grant_types <@ ARRAY['authorization_code', 'refresh_token']::text[]),
    ADD CONSTRAINT oauth_clients_response_types_allowed
        CHECK (cardinality(response_types) > 0 AND response_types <@ ARRAY['code']::text[]),
    ADD CONSTRAINT oauth_clients_access_ttl_range  CHECK (access_token_ttl  BETWEEN 300 AND 3600),
    ADD CONSTRAINT oauth_clients_refresh_ttl_range CHECK (refresh_token_ttl BETWEEN 3600 AND 2592000);

ALTER TABLE oauth_clients ALTER COLUMN access_token_ttl SET DEFAULT 3600;

COMMENT ON COLUMN oauth_clients.grant_types IS 'authorization_code and/or refresh_token. client_credentials is not supported.';
