-- Comment notifications must not unmask an anonymous creator, and nobody may
-- take the "Anonymous Creator" label as a pseud name.
--
-- 1. Migration 037's handle_comment_notifications trigger named the commenter
--    as COALESCE(pseud name, username, guest name) and put the commenter's
--    account id in the notification. When the commenter is a creator of a work
--    that hides its creators (anonymous, in an anonymous collection, or in a
--    collection that hasn't revealed them), the recipient could read their
--    pseud, username and account id. Now such a commenter is "Anonymous
--    Creator" with no id of any kind. (A guest's comment now notifies too:
--    "!=" against a missing user is never true, so guests were skipped.) The name is the pseud's name, then the
--    guest name, then "Someone"; never the username. No account id is put in
--    the payload at all.
--
-- 2. The comments API marks a hidden creator's comment as "Anonymous Creator".
--    A pseud with that name could look like one. Names are now reserved
--    (ignoring case and spaces): existing ones are renamed, and the table
--    refuses new ones. The default pseud made for a new account is renamed too
--    if the username would collide (the sign-up form also refuses those).

-- 2. Reserved name. Rename first so the constraint can be added.
UPDATE pseuds
SET name = left(name, 31) || '-' || left(replace(id::text, '-', ''), 8),
    updated_at = now()
WHERE lower(regexp_replace(name, '[[:space:]]', '', 'g')) = 'anonymouscreator';

ALTER TABLE pseuds ADD CONSTRAINT pseud_name_not_reserved
    CHECK (lower(regexp_replace(name, '[[:space:]]', '', 'g')) <> 'anonymouscreator');

CREATE OR REPLACE FUNCTION create_default_pseud()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO pseuds (user_id, name, is_default)
    VALUES (NEW.id,
            CASE WHEN lower(regexp_replace(NEW.username::text, '[[:space:]]', '', 'g')) = 'anonymouscreator'
                 THEN left(NEW.username::text, 31) || '-' || left(replace(NEW.id::text, '-', ''), 8)
                 ELSE left(NEW.username::text, 40) END,
            true)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1. Notifications.
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

    -- Who the commenter is called. A creator of a work that hides its
    -- creators is never named, and the username is never used.
    IF NEW.user_id IS NOT NULL AND work_hides_creator(NEW.work_id)
       AND is_work_creator(NEW.work_id, NEW.user_id) THEN
        commenter_name := 'Anonymous Creator';
    ELSE
        SELECT p.name INTO commenter_name FROM pseuds p WHERE p.id = NEW.pseudonym_id;
        commenter_name := COALESCE(commenter_name, NEW.guest_name, 'Someone');
    END IF;

    -- Handle mentions in comment content
    SELECT create_mentions(
        'comment',
        NEW.id,
        NEW.content,
        NEW.user_id
    ) INTO mention_count;

    -- Notify work author (if not commenting on their own work)
    IF work_author_id IS NOT NULL AND work_author_id IS DISTINCT FROM NEW.user_id THEN
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

    -- Handle reply notifications
    IF NEW.parent_comment_id IS NOT NULL THEN
        SELECT c.user_id INTO parent_comment_author_id
        FROM comments c
        WHERE c.id = NEW.parent_comment_id;

        -- Notify parent comment author (if not replying to themselves)
        IF parent_comment_author_id IS NOT NULL AND parent_comment_author_id IS DISTINCT FROM NEW.user_id THEN
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

-- 3. Mention notifications carried the mentioning account's id too. A hidden
--    creator who @-mentions someone in a comment would be unmasked by it. The
--    mention row keeps who wrote it for the server's use; the notification the
--    mentioned person reads says nothing about the author. (The old body also
--    compared a variable to the column of the same name and failed whenever a
--    comment mentioned a real user; the variables are renamed.)
CREATE OR REPLACE FUNCTION create_mentions(
    p_source_type VARCHAR(20),
    p_source_id UUID,
    p_content TEXT,
    p_mentioning_user_id UUID DEFAULT NULL
) RETURNS INTEGER AS $$
DECLARE
    mention_record RECORD;
    v_mentioned_user_id UUID;
    created_count INTEGER := 0;
    v_notification_id UUID;
BEGIN
    -- Extract mentions from content
    FOR mention_record IN 
        SELECT * FROM extract_mentions(p_content)
    LOOP
        -- Find the mentioned user
        SELECT id INTO v_mentioned_user_id 
        FROM users 
        WHERE username = mention_record.username;
        
        IF v_mentioned_user_id IS NOT NULL THEN
            -- Create mention record
            INSERT INTO mentions (
                source_type, source_id, mentioned_user_id, mentioning_user_id,
                content_excerpt, position_start, position_end
            ) VALUES (
                p_source_type, p_source_id, v_mentioned_user_id, p_mentioning_user_id,
                SUBSTRING(p_content FROM mention_record.position_start FOR 100),
                mention_record.position_start, mention_record.position_end
            );
            
            -- Create notification for the mention
            SELECT create_notification(
                v_mentioned_user_id,
                CASE p_source_type 
                    WHEN 'comment' THEN 'comment_mention'
                    WHEN 'work' THEN 'work_mention'
                    ELSE 'user_mention'
                END,
                'You were mentioned',
                CASE p_source_type
                    WHEN 'comment' THEN 'You were mentioned in a comment'
                    WHEN 'work' THEN 'You were mentioned in a work'
                    ELSE 'You were mentioned'
                END,
                jsonb_build_object(
                    'source_type', p_source_type,
                    'source_id', p_source_id,
                    'excerpt', SUBSTRING(p_content FROM mention_record.position_start FOR 100)
                )
            ) INTO v_notification_id;
            
            -- Link the mention to the notification
            UPDATE mentions 
            SET notification_id = v_notification_id 
            WHERE mentions.mentioned_user_id = v_mentioned_user_id 
              AND source_id = p_source_id 
              AND source_type = p_source_type;
            
            created_count := created_count + 1;
        END IF;
    END LOOP;
    
    RETURN created_count;
END;
$$ LANGUAGE plpgsql;
