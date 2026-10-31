-- Bookmarks and author subscriptions are tied to a pseud, not shown by account.
--
-- Bookmarks. AO3's bookmarks belong to the pseud they were made under
-- (app/models/bookmark.rb: belongs_to :pseud), which is how a pseud's page
-- lists its bookmarks. user_id stays: it says whose bookmark it is for
-- privacy checks and is never shown.
ALTER TABLE bookmarks ADD COLUMN IF NOT EXISTS pseud_id UUID REFERENCES pseuds(id) ON DELETE SET NULL;

UPDATE bookmarks b
SET pseud_id = (SELECT p.id FROM pseuds p WHERE p.user_id = b.user_id AND p.is_default)
WHERE b.pseud_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_bookmarks_pseud_id ON bookmarks(pseud_id, created_at DESC);

-- A bookmark made without saying which pseud is made under the default one.
CREATE OR REPLACE FUNCTION set_bookmark_pseud()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.pseud_id IS NULL THEN
        SELECT p.id INTO NEW.pseud_id FROM pseuds p WHERE p.user_id = NEW.user_id AND p.is_default;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_bookmark_pseud_trigger ON bookmarks;
CREATE TRIGGER set_bookmark_pseud_trigger
    BEFORE INSERT ON bookmarks
    FOR EACH ROW EXECUTE FUNCTION set_bookmark_pseud();

-- Author subscriptions. They used the author's account id as target_id and
-- the username as target_name, so the API handed out both. A subscription to
-- an author now follows one pseud: target_id is the pseud's id (public already,
-- like the pseud's name). Existing rows move to the account's default pseud.
-- Following one pseud and not the account means a subscriber learns nothing
-- about which other pseuds belong to the same person.
UPDATE subscriptions s
SET target_id = p.id, target_name = p.name, updated_at = now()
FROM pseuds p
WHERE s.type = 'author' AND p.user_id = s.target_id AND p.is_default;

UPDATE content_subscriptions s
SET target_id = p.id, updated_at = now()
FROM pseuds p
WHERE s.target_type = 'author' AND p.user_id = s.target_id AND p.is_default;

-- Whatever still points at no pseud (the account is gone) is dropped.
DELETE FROM subscriptions s
WHERE s.type = 'author' AND NOT EXISTS (SELECT 1 FROM pseuds p WHERE p.id = s.target_id);
DELETE FROM content_subscriptions s
WHERE s.target_type = 'author' AND NOT EXISTS (SELECT 1 FROM pseuds p WHERE p.id = s.target_id);
