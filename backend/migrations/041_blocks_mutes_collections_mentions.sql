-- Blocks, mutes, collection organisers and @mentions.
--
-- 1. Blocks are between two accounts, work in both directions across every
--    pseud, and leave works readable. Only comments (and kudos and gifts)
--    close. The blocker picks a pseud to block; that pseud is remembered only
--    so the blocker's own list can show what they picked.
-- 2. Mutes are per pseud, so a mute can never tell the muter that two pseuds
--    belong to the same person.
-- 3. Collections name their organiser by pseud, as AO3 does
--    (collection_participants.pseud_id), and not by account.
-- 4. AO3 has no @mentions, and the rebuild's version looked an account up by
--    name. It is removed.
--
-- AO3 references for the block rules:
--   app/models/block.rb (a block is between two users)
--   app/validators/not_blocked_validator.rb:18-21 (you are never blocked from
--     your own things; otherwise a block between you and any creator applies)
--   app/models/comment.rb:40-55 (no comment on a blocker's work; no reply to
--     a blocker's comment)
--   app/models/kudo.rb:14 (no kudos on a blocker's work)
--   app/models/work.rb:194-216 (a work can't be gifted to someone who blocked
--     one of its creators)
-- AO3 blocks one way only. The owner of this project asked for both ways, and
-- for a blocked reader to see comments as disabled, so those two rules are
-- stricter than AO3's.

-- ---------------------------------------------------------------------
-- 4. @mentions
-- ---------------------------------------------------------------------

-- Same as migration 039's version, minus the mention lookup, and with one
-- addition: a recipient who has blocked the commenter is not notified.
CREATE OR REPLACE FUNCTION handle_comment_notifications()
RETURNS TRIGGER AS $$
DECLARE
    work_author_id UUID;
    parent_comment_author_id UUID;
    notification_title TEXT;
    notification_message TEXT;
    work_title TEXT;
    commenter_name TEXT;
BEGIN
    SELECT w.user_id, w.title INTO work_author_id, work_title
    FROM works w
    WHERE w.id = NEW.work_id;

    -- Who the commenter is called. A creator of a work that hides its
    -- creators is never named, and the username is never used.
    IF NEW.user_id IS NOT NULL AND work_hides_creator(NEW.work_id)
       AND is_work_creator(NEW.work_id, NEW.user_id) THEN
        commenter_name := 'Anonymous Creator';
    ELSE
        SELECT p.name INTO commenter_name FROM pseuds p WHERE p.id = NEW.pseudonym_id;
        commenter_name := COALESCE(commenter_name, NEW.guest_name, 'Someone');
    END IF;

    -- Notify the work's author (if not commenting on their own work, and the
    -- author hasn't blocked the commenter).
    IF work_author_id IS NOT NULL AND work_author_id IS DISTINCT FROM NEW.user_id
       AND NOT (NEW.user_id IS NOT NULL AND EXISTS (
            SELECT 1 FROM user_blocks b
            WHERE b.blocker_id = work_author_id AND b.blocked_id = NEW.user_id)) THEN
        notification_title := format('New comment on "%s"', work_title);
        notification_message := format('%s commented on your work', commenter_name);

        PERFORM create_notification(
            work_author_id,
            'work_comment',
            notification_title,
            notification_message,
            jsonb_build_object(
                'work_id', NEW.work_id,
                'comment_id', NEW.id,
                'commenter_name', commenter_name,
                'work_title', work_title
            )
        );
    END IF;

    -- Notify the author of the comment being replied to.
    IF NEW.parent_comment_id IS NOT NULL THEN
        SELECT c.user_id INTO parent_comment_author_id
        FROM comments c
        WHERE c.id = NEW.parent_comment_id;

        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id IS DISTINCT FROM NEW.user_id
           AND NOT (NEW.user_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM user_blocks b
                WHERE b.blocker_id = parent_comment_author_id AND b.blocked_id = NEW.user_id)) THEN
            notification_title := 'Reply to your comment';
            notification_message := format('%s replied to your comment', commenter_name);

            PERFORM create_notification(
                parent_comment_author_id,
                'comment_reply',
                notification_title,
                notification_message,
                jsonb_build_object(
                    'work_id', NEW.work_id,
                    'comment_id', NEW.id,
                    'parent_comment_id', NEW.parent_comment_id,
                    'commenter_name', commenter_name,
                    'work_title', work_title
                )
            );
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP FUNCTION IF EXISTS create_mentions(VARCHAR, UUID, TEXT, UUID);
DROP FUNCTION IF EXISTS extract_mentions(TEXT);
DROP TABLE IF EXISTS mentions;

-- Notifications already sent for the removed feature.
DELETE FROM notifications WHERE type IN ('comment_mention', 'work_mention', 'user_mention');

-- ---------------------------------------------------------------------
-- 1. Blocks
-- ---------------------------------------------------------------------

-- The pseud the blocker picked. There is no foreign key on purpose: if the
-- blocked person deletes that pseud the block must stay, and the blocker's
-- list keeps showing the name they picked (pseud_name is a copy of it).
-- The block is keyed by (blocker, pseud) so blocking a second pseud of an
-- account that is already blocked just adds a row; the answer never tells the
-- blocker whether two pseuds belong to the same account.
ALTER TABLE user_blocks ADD COLUMN IF NOT EXISTS pseud_id UUID;
ALTER TABLE user_blocks ADD COLUMN IF NOT EXISTS pseud_name VARCHAR(40);

-- Blocks made before this migration were by account: show the account's
-- default pseud.
UPDATE user_blocks b
SET pseud_id = p.id, pseud_name = left(p.name, 40)
FROM pseuds p
WHERE p.user_id = b.blocked_id AND p.is_default AND b.pseud_id IS NULL;

DELETE FROM user_blocks WHERE pseud_id IS NULL;

ALTER TABLE user_blocks ALTER COLUMN pseud_id SET NOT NULL;
ALTER TABLE user_blocks ALTER COLUMN pseud_name SET NOT NULL;

DO $$
DECLARE
    c RECORD;
BEGIN
    FOR c IN SELECT conname FROM pg_constraint
             WHERE conrelid = 'user_blocks'::regclass AND contype = 'u' LOOP
        EXECUTE format('ALTER TABLE user_blocks DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;

-- AO3 has one kind of block. A block that was "comments" or "works" only
-- becomes a full block (the safe direction), and the reason is dropped.
ALTER TABLE user_blocks DROP COLUMN IF EXISTS block_type;
ALTER TABLE user_blocks DROP COLUMN IF EXISTS reason;

ALTER TABLE user_blocks ADD CONSTRAINT user_blocks_blocker_pseud_key UNIQUE (blocker_id, pseud_id);
CREATE INDEX IF NOT EXISTS idx_user_blocks_pair ON user_blocks(blocker_id, blocked_id);

COMMENT ON TABLE user_blocks IS 'A block between two accounts, in both directions across all pseuds. pseud_id is the pseud the blocker picked and only feeds the blocker''s own list.';

-- True when a block exists between the two accounts, in either direction.
CREATE OR REPLACE FUNCTION accounts_blocked(a UUID, b UUID)
RETURNS BOOLEAN AS $$
    SELECT a IS NOT NULL AND b IS NOT NULL AND a <> b AND EXISTS (
        SELECT 1 FROM user_blocks
        WHERE (blocker_id = a AND blocked_id = b) OR (blocker_id = b AND blocked_id = a))
$$ LANGUAGE sql STABLE;

-- True when the viewer can't interact (comment, reply, give kudos) with the
-- work because of a block: a block exists between the viewer and the account
-- of one of its creators. A creator is never blocked from their own work
-- (not_blocked_validator.rb:18-19), and admins are not subject to blocks.
-- The work is still readable; this only closes comments and kudos.
CREATE OR REPLACE FUNCTION blocked_from_work(work_uuid UUID, viewer_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT viewer_uuid IS NOT NULL
        AND NOT is_work_creator(work_uuid, viewer_uuid)
        AND NOT is_site_moderator(viewer_uuid)
        AND EXISTS (
            SELECT 1 FROM creatorships c JOIN pseuds p ON p.id = c.pseud_id
            WHERE c.creation_id = work_uuid AND c.creation_type = 'Work' AND c.approved
              AND accounts_blocked(p.user_id, viewer_uuid))
$$ LANGUAGE sql STABLE;

-- ---------------------------------------------------------------------
-- 2. Mutes are per pseud
-- ---------------------------------------------------------------------

ALTER TABLE user_mutes ADD COLUMN IF NOT EXISTS muted_pseud_id UUID REFERENCES pseuds(id) ON DELETE CASCADE;

-- An account-level mute becomes a mute of that account's default pseud.
UPDATE user_mutes m
SET muted_pseud_id = p.id
FROM pseuds p
WHERE p.user_id = m.muted_id AND p.is_default AND m.muted_pseud_id IS NULL;

DELETE FROM user_mutes WHERE muted_pseud_id IS NULL;

DROP FUNCTION IF EXISTS is_user_muted(UUID, UUID);
DROP FUNCTION IF EXISTS get_user_muted_list(UUID);

-- Dropping the column also drops the old unique key and the self-mute check.
ALTER TABLE user_mutes DROP COLUMN muted_id;
ALTER TABLE user_mutes ALTER COLUMN muted_pseud_id SET NOT NULL;
ALTER TABLE user_mutes ADD CONSTRAINT user_mutes_muter_pseud_key UNIQUE (muter_id, muted_pseud_id);
CREATE INDEX IF NOT EXISTS idx_user_mutes_muted_pseud ON user_mutes(muted_pseud_id);

COMMENT ON TABLE user_mutes IS 'A muter hides one pseud''s works and comments from their own view. It never covers the other pseuds of that account.';

-- ---------------------------------------------------------------------
-- Visibility
-- ---------------------------------------------------------------------

-- Works. Same as migration 033 except:
--   * a block no longer hides a work (the blocked reader can still find and
--     read it; hiding it would tell them which pseuds are linked);
--   * a mute is by pseud, and does not apply to a work that hides its
--     creators, since hiding it would tell the muter who wrote it.
CREATE OR REPLACE FUNCTION can_user_view_work(work_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
    is_author BOOLEAN := false;
BEGIN
    SELECT restricted, restricted_to_users, restricted_to_adults, status, user_id,
           is_anonymous, in_anon_collection, hidden_by_admin, spam
    INTO work_record
    FROM works
    WHERE id = work_uuid;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    is_author := is_work_creator(work_uuid, viewer_uuid);

    IF work_record.status = 'draft' THEN
        RETURN is_author;
    END IF;

    IF work_record.hidden_by_admin OR work_record.spam THEN
        RETURN is_author OR is_site_moderator(viewer_uuid);
    END IF;

    IF is_author THEN
        RETURN true;
    END IF;

    IF (work_record.restricted OR work_record.restricted_to_users) AND viewer_uuid IS NULL THEN
        RETURN false;
    END IF;

    IF viewer_uuid IS NOT NULL THEN
        IF is_site_moderator(viewer_uuid) THEN
            RETURN true;
        END IF;

        IF NOT work_hides_creator(work_uuid) AND EXISTS (
            SELECT 1 FROM user_mutes um
            JOIN creatorships c ON c.pseud_id = um.muted_pseud_id
            WHERE um.muter_id = viewer_uuid
              AND c.creation_id = work_uuid
              AND c.creation_type = 'Work'
              AND c.approved
        ) THEN
            RETURN false;
        END IF;
    END IF;

    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Comments. Same as migration 033 plus:
--   * on a work the viewer is blocked from, there are no comments at all;
--   * the blocker never sees a comment written by an account they blocked,
--     anywhere (and, as with spam, replies under it go with it);
--   * a muter doesn't see comments written under a pseud they muted, unless
--     the comment is by a hidden creator (it shows as "Anonymous Creator",
--     so hiding it would give the creator away).
CREATE OR REPLACE FUNCTION can_user_view_comment(comment_uuid uuid, viewer_uuid uuid DEFAULT NULL::uuid)
RETURNS BOOLEAN AS $$
DECLARE
    c RECORD;
    the_work uuid;
BEGIN
    SELECT work_id, chapter_id, parent_comment_id, user_id, pseudonym_id, is_deleted, is_spam, unreviewed
    INTO c FROM comments WHERE id = comment_uuid;
    IF NOT FOUND OR c.is_deleted THEN
        RETURN false;
    END IF;
    the_work := c.work_id;
    IF c.chapter_id IS NOT NULL THEN
        IF NOT can_user_view_chapter(c.chapter_id, viewer_uuid) THEN
            RETURN false;
        END IF;
        SELECT work_id INTO the_work FROM chapters WHERE id = c.chapter_id;
    ELSIF NOT can_user_view_work(the_work, viewer_uuid) THEN
        RETURN false;
    END IF;
    IF c.is_spam AND NOT is_site_moderator(viewer_uuid) THEN
        RETURN false;
    END IF;
    IF viewer_uuid IS NOT NULL THEN
        IF blocked_from_work(the_work, viewer_uuid) THEN
            RETURN false;
        END IF;
        IF NOT is_site_moderator(viewer_uuid) THEN
            IF c.user_id IS NOT NULL AND EXISTS (
                   SELECT 1 FROM user_blocks b
                   WHERE b.blocker_id = viewer_uuid AND b.blocked_id = c.user_id) THEN
                RETURN false;
            END IF;
            IF c.pseudonym_id IS NOT NULL
               AND EXISTS (SELECT 1 FROM user_mutes m
                           WHERE m.muter_id = viewer_uuid AND m.muted_pseud_id = c.pseudonym_id)
               AND NOT (c.user_id IS NOT NULL AND work_hides_creator(the_work)
                        AND is_work_creator(the_work, c.user_id)) THEN
                RETURN false;
            END IF;
        END IF;
    END IF;
    IF c.unreviewed AND NOT (
           is_work_creator(the_work, viewer_uuid)
           OR is_site_moderator(viewer_uuid)
           OR (viewer_uuid IS NOT NULL AND c.user_id = viewer_uuid)) THEN
        RETURN false;
    END IF;
    IF c.parent_comment_id IS NOT NULL THEN
        RETURN can_user_view_comment(c.parent_comment_id, viewer_uuid);
    END IF;
    RETURN true;
END;
$$ LANGUAGE plpgsql STABLE;

-- Older helper, no longer used by the application; kept correct. Its block
-- and mute checks used account ids and block types that are gone.
CREATE OR REPLACE FUNCTION can_user_comment_on_work(work_uuid UUID, commenter_uuid UUID DEFAULT NULL)
RETURNS BOOLEAN AS $$
DECLARE
    work_record RECORD;
BEGIN
    SELECT comment_policy, disable_comments INTO work_record FROM works WHERE id = work_uuid;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    IF work_record.disable_comments = true OR work_record.comment_policy = 'disabled' THEN
        RETURN false;
    END IF;
    IF work_record.comment_policy = 'users_only' AND commenter_uuid IS NULL THEN
        RETURN false;
    END IF;
    RETURN NOT blocked_from_work(work_uuid, commenter_uuid);
END;
$$ LANGUAGE plpgsql STABLE;

-- ---------------------------------------------------------------------
-- 3. Collections name their organiser by pseud
-- ---------------------------------------------------------------------

-- The pseud the collection's owner shows as. user_id stays: it says who may
-- change the collection and is never shown.
ALTER TABLE collections ADD COLUMN IF NOT EXISTS pseud_id UUID REFERENCES pseuds(id) ON DELETE SET NULL;

UPDATE collections c
SET pseud_id = p.id
FROM pseuds p
WHERE p.user_id = c.user_id AND p.is_default AND c.pseud_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_collections_pseud_id ON collections(pseud_id);

-- A collection made without saying which pseud is made under the default one.
CREATE OR REPLACE FUNCTION set_collection_pseud()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.pseud_id IS NULL THEN
        SELECT p.id INTO NEW.pseud_id FROM pseuds p WHERE p.user_id = NEW.user_id AND p.is_default;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_collection_pseud_trigger ON collections;
CREATE TRIGGER set_collection_pseud_trigger
    BEFORE INSERT ON collections
    FOR EACH ROW EXECUTE FUNCTION set_collection_pseud();
