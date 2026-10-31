-- One pseud table, following AO3's pseud model.
--
-- Until now there were two: pseuds (authorship: creatorships, gifts) and
-- user_pseudonyms (comments, and where descriptions were written). This
-- migration makes pseuds the only table, moves the rows of user_pseudonyms
-- into it, points comments at it, and drops user_pseudonyms.
--
-- AO3 rules (app/models/pseud.rb): a name is 1 to 40 characters, unique per
-- user ignoring case; a user has exactly one default pseud; a description is
-- at most 500 characters; icon alt text 250 and icon comment text 50.
-- The characters a name may use (letters, digits, marks, spaces, underscores
-- and dashes, in any script) are checked by the app (shared/models/pseud.go),
-- because Postgres's idea of a letter depends on how the database was created.

-- ---------------------------------------------------------------------------
-- 1. The columns AO3 has that pseuds lacked. The icon file itself is not
--    stored here: icon_url says where it is.
-- ---------------------------------------------------------------------------

ALTER TABLE pseuds ADD COLUMN IF NOT EXISTS icon_url VARCHAR(500);
ALTER TABLE pseuds ADD COLUMN IF NOT EXISTS icon_alt_text VARCHAR(250) NOT NULL DEFAULT '';
ALTER TABLE pseuds ADD COLUMN IF NOT EXISTS icon_comment_text VARCHAR(50) NOT NULL DEFAULT '';

-- Names may now hold spaces and any script, and may differ by case (the
-- unique-per-user rule is put back, case-insensitively, at the end).
ALTER TABLE pseuds DROP CONSTRAINT IF EXISTS pseud_name_format;
ALTER TABLE pseuds DROP CONSTRAINT IF EXISTS pseuds_user_id_name_key;

UPDATE pseuds SET is_default = false WHERE is_default IS NULL;
ALTER TABLE pseuds ALTER COLUMN is_default SET NOT NULL;
ALTER TABLE pseuds ALTER COLUMN is_default SET DEFAULT false;

-- ---------------------------------------------------------------------------
-- 2. Names that differ only by case would break the new unique rule. Rename
--    the later ones (they keep their id, so nothing that points at them moves).
-- ---------------------------------------------------------------------------

WITH ranked AS (
    SELECT id, row_number() OVER (
        PARTITION BY user_id, lower(name) ORDER BY is_default DESC, created_at, id) AS n
    FROM pseuds
)
UPDATE pseuds p
SET name = left(p.name, 31) || '-' || left(replace(p.id::text, '-', ''), 8)
FROM ranked r
WHERE r.id = p.id AND r.n > 1;

-- ---------------------------------------------------------------------------
-- 3. Work out where each user_pseudonyms row goes.
--    - A row with the same user and the same name (ignoring case) as a pseud
--      is that pseud: new_id is the pseud's id.
--    - Any other row becomes a new pseud that keeps its own id, unless that
--      id is already taken by a pseud, in which case it gets a fresh one.
--    - Names that don't fit (over 40 characters, or blank) are shortened, and
--      given a piece of the id if that makes them collide.
-- ---------------------------------------------------------------------------

CREATE TEMPORARY TABLE pseud_map ON COMMIT DROP AS
WITH old AS (
    SELECT up.id AS old_id, up.user_id, up.name::text AS name, up.is_default AS was_default,
           up.description, up.icon_url, up.created_at,
           left(coalesce(nullif(btrim(up.name::text), ''), 'pseud'), 40) AS fitted
    FROM user_pseudonyms up
),
matched AS (
    SELECT o.*, p.id AS new_id, true AS existing
    FROM old o
    JOIN pseuds p ON p.user_id = o.user_id AND lower(p.name) = lower(o.name)
),
fresh AS (
    SELECT o.*,
           row_number() OVER (PARTITION BY o.user_id, lower(o.fitted) ORDER BY o.created_at, o.old_id) AS n,
           EXISTS (SELECT 1 FROM pseuds p WHERE p.user_id = o.user_id AND lower(p.name) = lower(o.fitted)) AS taken
    FROM old o
    WHERE o.old_id NOT IN (SELECT old_id FROM matched)
)
SELECT old_id, new_id, user_id, name, was_default, description, icon_url, created_at, existing,
       name AS final_name
FROM matched
UNION ALL
SELECT old_id,
       CASE WHEN EXISTS (SELECT 1 FROM pseuds p WHERE p.id = f.old_id) THEN uuid_generate_v4() ELSE f.old_id END,
       user_id, name, was_default, description, icon_url, created_at, false,
       CASE WHEN f.n > 1 OR f.taken
            THEN left(f.fitted, 31) || '-' || left(replace(f.old_id::text, '-', ''), 8)
            ELSE f.fitted END
FROM fresh f;

-- Pseuds that already exist take a description and icon from the matching row
-- if they don't have their own.
UPDATE pseuds p
SET description = coalesce(nullif(p.description, ''), m.description),
    icon_url = coalesce(p.icon_url, m.icon_url)
FROM pseud_map m
WHERE m.existing AND m.new_id = p.id;

INSERT INTO pseuds (id, user_id, name, description, icon_url, is_default, created_at, updated_at)
SELECT new_id, user_id, final_name, description, icon_url, false, created_at, created_at
FROM pseud_map
WHERE NOT existing;

-- ---------------------------------------------------------------------------
-- 4. Point everything that referenced user_pseudonyms at pseuds.
--    Only comments and comment_kudos did. The comment triggers are switched
--    off for the update: nothing about the comments changes except whose
--    pseud row they name.
-- ---------------------------------------------------------------------------

ALTER TABLE comments DROP CONSTRAINT IF EXISTS comments_pseudonym_id_fkey;
ALTER TABLE comment_kudos DROP CONSTRAINT IF EXISTS comment_kudos_pseudonym_id_fkey;

ALTER TABLE comments DISABLE TRIGGER USER;
UPDATE comments c SET pseudonym_id = m.new_id
FROM pseud_map m WHERE c.pseudonym_id = m.old_id AND m.old_id <> m.new_id;
ALTER TABLE comments ENABLE TRIGGER USER;

UPDATE comment_kudos k SET pseudonym_id = m.new_id
FROM pseud_map m WHERE k.pseudonym_id = m.old_id AND m.old_id <> m.new_id;

ALTER TABLE comments ADD CONSTRAINT comments_pseudonym_id_fkey
    FOREIGN KEY (pseudonym_id) REFERENCES pseuds(id);
ALTER TABLE comment_kudos ADD CONSTRAINT comment_kudos_pseudonym_id_fkey
    FOREIGN KEY (pseudonym_id) REFERENCES pseuds(id) ON DELETE SET NULL;

-- ---------------------------------------------------------------------------
-- 5. Exactly one default pseud per user.
--    More than one: keep the oldest. None: take the pseud that was the
--    default in user_pseudonyms, else the one named after the username, else
--    the oldest. A user with no pseud at all gets one named after the username.
-- ---------------------------------------------------------------------------

UPDATE pseuds SET is_default = false
WHERE id IN (
    SELECT id FROM (
        SELECT id, row_number() OVER (PARTITION BY user_id ORDER BY created_at, id) AS n
        FROM pseuds WHERE is_default
    ) t WHERE n > 1);

UPDATE pseuds SET is_default = true
WHERE id IN (
    SELECT DISTINCT ON (p.user_id) p.id
    FROM pseuds p
    JOIN users u ON u.id = p.user_id
    LEFT JOIN pseud_map m ON m.new_id = p.id AND m.was_default
    WHERE NOT EXISTS (SELECT 1 FROM pseuds d WHERE d.user_id = p.user_id AND d.is_default)
    ORDER BY p.user_id, (m.new_id IS NOT NULL) DESC,
             (lower(p.name) = lower(u.username::text)) DESC, p.created_at, p.id);

INSERT INTO pseuds (user_id, name, is_default)
SELECT u.id, left(u.username::text, 40), true
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM pseuds p WHERE p.user_id = u.id);

-- ---------------------------------------------------------------------------
-- 6. Views and functions that read user_pseudonyms. The two views were not
--    used by the app (and showed more of a user than the API may), so they go.
-- ---------------------------------------------------------------------------

DROP VIEW IF EXISTS comments_with_details;
DROP VIEW IF EXISTS user_profiles;

CREATE OR REPLACE FUNCTION handle_comment_notifications()
RETURNS TRIGGER AS $$
DECLARE
    work_author_id UUID;
    parent_comment_author_id UUID;
    notification_title TEXT;
    notification_message TEXT;
    work_title TEXT;
    commenter_name TEXT;
    mention_count INTEGER;
BEGIN
    -- Get work information
    SELECT w.user_id, w.title INTO work_author_id, work_title
    FROM works w
    WHERE w.id = NEW.work_id;

    -- Get commenter name
    SELECT COALESCE(p.name, u.username, NEW.guest_name) INTO commenter_name
    FROM users u
    LEFT JOIN pseuds p ON NEW.pseudonym_id = p.id
    WHERE u.id = NEW.user_id;

    -- Handle mentions in comment content
    SELECT create_mentions(
        'comment',
        NEW.id,
        NEW.content,
        NEW.user_id
    ) INTO mention_count;

    -- Notify work author (if not commenting on their own work)
    IF work_author_id IS NOT NULL AND work_author_id != NEW.user_id THEN
        notification_title := format('New comment on "%s"', work_title);
        notification_message := format('%s commented on your work', COALESCE(commenter_name, 'Someone'));

        PERFORM create_notification(
            work_author_id,
            'work_comment',
            notification_title,
            notification_message,
            jsonb_build_object(
                'work_id', NEW.work_id,
                'comment_id', NEW.id,
                'commenter_id', NEW.user_id,
                'commenter_name', commenter_name,
                'work_title', work_title
            )
        );
    END IF;

    -- Handle reply notifications
    IF NEW.parent_comment_id IS NOT NULL THEN
        -- Get parent comment author
        SELECT c.user_id INTO parent_comment_author_id
        FROM comments c
        WHERE c.id = NEW.parent_comment_id;

        -- Notify parent comment author (if not replying to themselves)
        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id != NEW.user_id THEN
            notification_title := 'Reply to your comment';
            notification_message := format('%s replied to your comment', COALESCE(commenter_name, 'Someone'));

            PERFORM create_notification(
                parent_comment_author_id,
                'comment_reply',
                notification_title,
                notification_message,
                jsonb_build_object(
                    'work_id', NEW.work_id,
                    'comment_id', NEW.id,
                    'parent_comment_id', NEW.parent_comment_id,
                    'commenter_id', NEW.user_id,
                    'commenter_name', commenter_name,
                    'work_title', work_title
                )
            );
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- get_work_authors names a work's creators for the API. A work that hides its
-- creators (anonymous, in an anonymous collection, or in a collection that
-- hasn't revealed them yet) names nobody, whoever asks. The columns stay the
-- same so callers don't change; the API only passes on the pseud's id and name.
CREATE OR REPLACE FUNCTION get_work_authors(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS TABLE(
    pseud_id UUID,
    pseud_name VARCHAR,
    user_id UUID,
    username CITEXT,
    is_anonymous BOOLEAN
) AS $$
DECLARE
    hidden BOOLEAN;
BEGIN
    SELECT coalesce(w.is_anonymous, false) OR coalesce(w.in_anon_collection, false)
           OR coalesce(w.in_unrevealed_collection, false)
    INTO hidden
    FROM works w WHERE w.id = work_uuid;

    IF hidden THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT p.id, p.name, p.user_id, u.username, false
    FROM creatorships c
    JOIN pseuds p ON c.pseud_id = p.id
    JOIN users u ON p.user_id = u.id
    WHERE c.creation_id = work_uuid
    AND c.creation_type = 'Work'
    AND c.approved = true
    ORDER BY c.created_at;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- 7. Drop the old table and put AO3's rules on pseuds.
-- ---------------------------------------------------------------------------

DROP TABLE user_pseudonyms;

-- A default pseud can no longer be deleted by promoting another one first: the
-- app refuses to delete the default, and that trigger would now collide with
-- the one-default-per-user index.
DROP TRIGGER IF EXISTS ensure_default_pseud_trigger ON pseuds;
DROP FUNCTION IF EXISTS ensure_default_pseud();

DROP INDEX IF EXISTS idx_pseuds_default;
DROP INDEX IF EXISTS idx_pseuds_name;

-- Unique per user, ignoring case (AO3's collation does the same).
CREATE UNIQUE INDEX pseuds_user_name_key ON pseuds (user_id, lower(name));
-- Exactly one default pseud per user.
CREATE UNIQUE INDEX pseuds_one_default_per_user ON pseuds (user_id) WHERE is_default;
-- Name lookups by prefix and by name.
CREATE INDEX idx_pseuds_lower_name ON pseuds (lower(name) text_pattern_ops);

ALTER TABLE pseuds ADD CONSTRAINT pseud_name_has_content
    CHECK (name ~ '[^ _-]' AND name !~ '[[:cntrl:]]');
ALTER TABLE pseuds ADD CONSTRAINT pseud_icon_alt_text_length CHECK (char_length(icon_alt_text) <= 250);
ALTER TABLE pseuds ADD CONSTRAINT pseud_icon_comment_text_length CHECK (char_length(icon_comment_text) <= 50);

-- Every account gets a default pseud named after its username, in the same
-- transaction as the account (AO3: User#create_default_associateds).
CREATE OR REPLACE FUNCTION create_default_pseud()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO pseuds (user_id, name, is_default)
    VALUES (NEW.id, left(NEW.username::text, 40), true)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_default_pseud_trigger
    AFTER INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION create_default_pseud();

COMMENT ON TABLE pseuds IS 'AO3 pseuds: the names users write, comment and are credited under. The only pseud table.';
