-- OAuth provider, phase 5: apps can now call the API with their tokens.
--
-- Adds the one scope the routes needed that did not exist (following
-- authors, works, series and tags), and makes the description of "read" say
-- what it now allows. The list the server reads is
-- shared/models/oauth.go (AO3OAuthScopes); a test checks this table agrees.

INSERT INTO oauth_scopes (name, description, category, is_default, requires_consent, is_admin_only)
VALUES ('subscriptions:manage', 'Follow and unfollow authors, works, series and tags for you', 'social', false, true, false)
ON CONFLICT (name) DO NOTHING;

UPDATE oauth_scopes
   SET description = 'Read works, series, collections, comments, bookmarks and tags as you see them (including works only members can see), and what you follow'
 WHERE name = 'read';
