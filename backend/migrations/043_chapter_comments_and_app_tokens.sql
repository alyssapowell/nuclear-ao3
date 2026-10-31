-- Two schema rules that the code already assumed.

-- ---------------------------------------------------------------------
-- 1. Comments on a chapter
-- ---------------------------------------------------------------------
--
-- Migration 002 said a comment belongs to a work OR a chapter, never both
-- (check constraint comment_target). The code never worked that way:
-- CreateComment writes both work_id and chapter_id for a chapter comment, so
-- every comment on a chapter was refused by the database.
--
-- AO3 (app/models/comment.rb): a comment's "commentable" is a chapter for
-- comments on a work (commentable_object turns a Work into its last posted
-- chapter), a comment for a reply, and its ultimate_parent is always the work.
-- Everything about a comment (who may see it, the work's comment count, who
-- is notified, blocks, moderation) goes through that work.
--
-- So the rule here is: every comment belongs to a work (work_id), and may
-- also name the chapter it was left on (chapter_id), which has to be a chapter
-- of that same work. A comment with no chapter is a comment on the work as a
-- whole. The work's counters, the notification trigger and the visibility
-- functions all read work_id, which is why it is always kept.

ALTER TABLE comments DROP CONSTRAINT comment_target;

-- Rows that named only a chapter (nothing in the code wrote them, but 002
-- allowed them) get that chapter's work. The counters are rebuilt after, with
-- the per-row triggers switched off as the cleanup job does.
SET LOCAL nuclear.maintenance = 'on';
UPDATE comments c SET work_id = ch.work_id
FROM chapters ch
WHERE c.work_id IS NULL AND c.chapter_id = ch.id;
UPDATE works w SET comment_count = (
    SELECT COUNT(*) FROM comments c
    WHERE c.work_id = w.id AND NOT c.is_deleted AND NOT c.shadowed)
WHERE EXISTS (SELECT 1 FROM comments c WHERE c.work_id = w.id AND c.chapter_id IS NOT NULL);
SET LOCAL nuclear.maintenance = 'off';

ALTER TABLE comments ALTER COLUMN work_id SET NOT NULL;

-- The chapter has to belong to the comment's work. A composite foreign key
-- says so without a trigger; it is not checked when chapter_id is empty.
ALTER TABLE chapters ADD CONSTRAINT chapters_work_id_id_key UNIQUE (work_id, id);
ALTER TABLE comments ADD CONSTRAINT comments_chapter_in_work
    FOREIGN KEY (work_id, chapter_id) REFERENCES chapters (work_id, id) ON DELETE CASCADE;

COMMENT ON COLUMN comments.chapter_id IS 'The chapter the comment was left on, if any. Always a chapter of work_id; work_id is always set.';

-- ---------------------------------------------------------------------
-- 2. OAuth tokens that belong to an app, not a person
-- ---------------------------------------------------------------------

-- A "client credentials" access token belongs to an app, so it has no user.
-- The column said NOT NULL, so every such grant failed with a server error
-- when the token was stored.
ALTER TABLE oauth_access_tokens ALTER COLUMN user_id DROP NOT NULL;
