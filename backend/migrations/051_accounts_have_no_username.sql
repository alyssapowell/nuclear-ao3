-- Accounts have no username. The email address is the login, and the public
-- face of an account is only its pseuds, which are never linked to each other
-- or to the account. Sign-up now asks for a pen name instead, and that is the
-- name of the account's first, default pseud.
--
-- The column stays, because an importer may want to keep the login an
-- account had elsewhere. It is optional now, free-form, and still UNIQUE so
-- an imported name can't collide with another. Nothing reads it for login, and
-- nothing shows it.
--
-- 1. The column no longer has to be filled in, or to look like anything.
-- 2. A new account has no username, so the trigger that made its default pseud
--    from the username leaves it alone; sign-up creates the pseud itself. An
--    imported account that does have a username still gets a pseud from it.
-- 3. get_work_authors stops returning the username column.
-- 4. The views that selected the username (some beside the author of a
--    work) are recreated without it.

-- 1.
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS username_format;

-- 2.
CREATE OR REPLACE FUNCTION create_default_pseud() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.username IS NULL THEN
        RETURN NEW;
    END IF;
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

-- 3.
DROP FUNCTION IF EXISTS get_work_authors(UUID, UUID);
CREATE FUNCTION get_work_authors(work_uuid UUID, viewer_uuid UUID DEFAULT NULL)
RETURNS TABLE(pseud_id UUID, pseud_name VARCHAR, user_id UUID, is_anonymous BOOLEAN) AS $$
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
    SELECT p.id, p.name, p.user_id, false
    FROM creatorships c
    JOIN pseuds p ON c.pseud_id = p.id
    WHERE c.creation_id = work_uuid
    AND c.creation_type = 'Work'
    AND c.approved = true
    ORDER BY c.created_at;
END;
$$ LANGUAGE plpgsql;

-- 4. A view can't lose a column with CREATE OR REPLACE, so each is dropped and
-- made again with the same columns, less the username.
DROP VIEW IF EXISTS active_users_with_roles;
CREATE VIEW active_users_with_roles AS
SELECT
    u.id,
    u.email,
    u.display_name,
    u.is_verified,
    u.last_login_at,
    u.created_at,
    array_agg(ur.role ORDER BY ur.role) FILTER (WHERE ur.role IS NOT NULL AND ur.revoked_at IS NULL) AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.revoked_at IS NULL
WHERE u.is_active = true
GROUP BY u.id, u.email, u.display_name, u.is_verified, u.last_login_at, u.created_at;

DROP VIEW IF EXISTS user_security_summary;
CREATE VIEW user_security_summary AS
SELECT
    u.id,
    u.email,
    u.last_login_at,
    count(DISTINCT rt.id) FILTER (WHERE rt.revoked_at IS NULL AND rt.expires_at > now()) AS active_refresh_tokens,
    count(DISTINCT us.id) FILTER (WHERE us.is_active = true) AS active_sessions,
    count(DISTINCT se.id) FILTER (WHERE se.created_at > (now() - '30 days'::interval)) AS recent_security_events,
    max(se.created_at) FILTER (WHERE se.severity::text = ANY (ARRAY['high', 'critical'])) AS last_high_severity_event
FROM users u
LEFT JOIN refresh_tokens rt ON u.id = rt.user_id
LEFT JOIN user_sessions us ON u.id = us.user_id
LEFT JOIN security_events se ON u.id = se.user_id
WHERE u.is_active = true
GROUP BY u.id, u.email, u.last_login_at;

DROP VIEW IF EXISTS user_active_tokens;
CREATE VIEW user_active_tokens AS
SELECT
    u.id AS user_id,
    u.email,
    c.client_name,
    at.scopes,
    at.created_at,
    at.last_used,
    at.expires_at
FROM oauth_access_tokens at
JOIN users u ON at.user_id = u.id
JOIN oauth_clients c ON at.client_id = c.client_id
WHERE at.is_revoked = false AND at.expires_at > now()
ORDER BY at.created_at DESC;

DROP VIEW IF EXISTS user_dashboard;
CREATE VIEW user_dashboard AS
SELECT
    u.id,
    u.display_name,
    count(DISTINCT n.id) FILTER (WHERE n.is_read = false) AS unread_notifications,
    count(DISTINCT w.id) FILTER (WHERE w.is_draft = false) AS published_works,
    count(DISTINCT w2.id) FILTER (WHERE w2.is_draft = true) AS draft_works,
    count(DISTINCT b.id) AS bookmarks,
    count(DISTINCT s.id) AS series,
    COALESCE(sum(w.hit_count), 0::bigint) AS total_hits,
    COALESCE(sum(w.kudos_count), 0::bigint) AS total_kudos,
    COALESCE(sum(w.comment_count), 0::bigint) AS total_comments,
    max(w.published_at) AS last_published
FROM users u
LEFT JOIN notifications n ON u.id = n.user_id
LEFT JOIN works w ON u.id = w.user_id AND w.is_draft = false
LEFT JOIN works w2 ON u.id = w2.user_id AND w2.is_draft = true
LEFT JOIN bookmarks b ON u.id = b.user_id
LEFT JOIN series s ON u.id = s.user_id
GROUP BY u.id, u.display_name;

DROP VIEW IF EXISTS popular_works;
CREATE VIEW popular_works AS
SELECT
    w.id, w.title, w.summary, w.notes, w.end_notes, w.user_id, w.language, w.rating,
    w.archive_warning, w.category, w.word_count, w.chapter_count, w.expected_chapters,
    w.hit_count, w.kudos_count, w.comment_count, w.bookmark_count, w.is_complete,
    w.is_draft, w.published_at, w.completed_at, w.imported_from, w.external_id,
    w.restricted, w.moderated, w.created_at, w.updated_at,
    u.display_name AS author_display_name,
    (w.kudos_count * 2 + w.comment_count * 3 + w.bookmark_count * 4)::numeric + w.hit_count::numeric * 0.1 AS popularity_score
FROM works w
JOIN users u ON w.user_id = u.id
WHERE w.is_draft = false AND w.published_at IS NOT NULL AND NOT w.hidden_by_admin AND NOT w.spam
ORDER BY ((w.kudos_count * 2 + w.comment_count * 3 + w.bookmark_count * 4)::numeric + w.hit_count::numeric * 0.1) DESC;

DROP VIEW IF EXISTS published_works_with_stats;
CREATE VIEW published_works_with_stats AS
SELECT
    w.id, w.title, w.summary, w.notes, w.end_notes, w.user_id, w.language, w.rating,
    w.archive_warning, w.category, w.word_count, w.chapter_count, w.expected_chapters,
    w.hit_count, w.kudos_count, w.comment_count, w.bookmark_count, w.is_complete,
    w.is_draft, w.published_at, w.completed_at, w.imported_from, w.external_id,
    w.restricted, w.moderated, w.created_at, w.updated_at, w.warnings, w.fandoms,
    w.characters, w.relationships, w.freeform_tags, w.max_chapters, w.status,
    u.display_name AS author_display_name,
    ws.hits, ws.kudos, ws.comments, ws.bookmarks,
    count(DISTINCT wt.tag_id) AS tag_count,
    array_agg(DISTINCT t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL) AS tag_names,
    array_agg(DISTINCT t.type ORDER BY t.type) FILTER (WHERE t.type IS NOT NULL) AS tag_types
FROM works w
JOIN users u ON w.user_id = u.id
LEFT JOIN work_statistics ws ON w.id = ws.work_id
LEFT JOIN work_tags wt ON w.id = wt.work_id
LEFT JOIN tags t ON wt.tag_id = t.id
WHERE w.is_draft = false AND w.published_at IS NOT NULL AND NOT w.hidden_by_admin AND NOT w.spam
GROUP BY w.id, u.display_name, ws.hits, ws.kudos, ws.comments, ws.bookmarks;
