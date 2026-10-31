-- The login signing key is now a file the admin keeps (SIGNING_KEY_FILE), not
-- a row in the database. Drop the table so the old plaintext private keys are
-- gone from the database and from future backups.
--
-- Existing logins end once: tokens signed with the old key stop working, and
-- everyone logs in again. (Old backups still hold the old key; destroy them
-- if you suspect it leaked.)
DROP TABLE IF EXISTS jwt_signing_keys;
