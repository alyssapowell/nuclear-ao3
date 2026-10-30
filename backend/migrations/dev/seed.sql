-- Sample data for local development. NEVER load this into a real archive:
-- every account here has the password "password123", and one is an admin.
--
-- Load it with:  make seed
-- It's safe to load more than once.

-- Accounts (all with password "password123")
INSERT INTO users (id, username, email, password_hash, display_name, is_verified) VALUES
    ('123e4567-e89b-12d3-a456-426614174000', 'testuser',    'testuser@example.org',    '$2a$12$KuU.9E1g38p/DCZAeprP/eocW9koHTL8.4jDrD7EqK.lds8kqm7vW', 'Test User', true),
    ('123e4567-e89b-12d3-a456-426614174001', 'author2',     'author2@example.org',     '$2a$12$KuU.9E1g38p/DCZAeprP/eocW9koHTL8.4jDrD7EqK.lds8kqm7vW', 'Famous Author', true),
    ('123e4567-e89b-12d3-a456-426614174002', 'tagwrangler', 'tagwrangler@example.org', '$2a$12$KuU.9E1g38p/DCZAeprP/eocW9koHTL8.4jDrD7EqK.lds8kqm7vW', 'Tag Wrangler', true),
    ('123e4567-e89b-12d3-a456-426614174003', 'admin',       'admin@example.org',       '$2a$12$KuU.9E1g38p/DCZAeprP/eocW9koHTL8.4jDrD7EqK.lds8kqm7vW', 'Site Administrator', true)
ON CONFLICT (username) DO NOTHING;

INSERT INTO user_roles (user_id, role) VALUES
    ('123e4567-e89b-12d3-a456-426614174002', 'tag_wrangler'),
    ('123e4567-e89b-12d3-a456-426614174003', 'admin')
ON CONFLICT (user_id, role) DO NOTHING;

INSERT INTO pseuds (user_id, name, is_default)
SELECT id, username, true FROM users
WHERE id::text LIKE '123e4567-e89b-12d3-a456-42661417400_'
  AND NOT EXISTS (SELECT 1 FROM pseuds p WHERE p.user_id = users.id AND p.is_default);

-- Tags
INSERT INTO tags (name, type, is_canonical) VALUES
    ('Harry Potter - J. K. Rowling', 'fandom', true),
    ('Sherlock Holmes & Related Fandoms', 'fandom', true),
    ('Marvel Cinematic Universe', 'fandom', true),
    ('Hermione Granger', 'character', true),
    ('Sherlock Holmes', 'character', true),
    ('John Watson', 'character', true),
    ('Sherlock Holmes/John Watson', 'relationship', true),
    ('Alternate Universe', 'freeform', true),
    ('Fluff', 'freeform', true),
    ('Angst', 'freeform', true)
ON CONFLICT (name) DO NOTHING;

-- Two posted works, each with one chapter
INSERT INTO works (id, title, summary, user_id, language, rating, fandoms, characters, relationships,
                   freeform_tags, chapter_count, word_count, is_complete, is_draft, status, published_at)
VALUES
    ('a0000000-0000-0000-0000-000000000001', 'The Brightest Witch of Her Age',
     'Hermione finds magic that goes beyond anything taught at Hogwarts.',
     '123e4567-e89b-12d3-a456-426614174000', 'en', 'teen',
     ARRAY['Harry Potter - J. K. Rowling'], ARRAY['Hermione Granger'], ARRAY[]::text[],
     ARRAY['Alternate Universe'], 1, 0, true, false, 'published', NOW() - INTERVAL '2 days'),
    ('a0000000-0000-0000-0000-000000000002', 'Baker Street Mysteries',
     'Cases that never made it into Dr. Watson''s published accounts.',
     '123e4567-e89b-12d3-a456-426614174001', 'en', 'general',
     ARRAY['Sherlock Holmes & Related Fandoms'], ARRAY['Sherlock Holmes', 'John Watson'],
     ARRAY['Sherlock Holmes/John Watson'], ARRAY['Fluff'], 1, 0, false, false, 'published', NOW() - INTERVAL '1 week')
ON CONFLICT (id) DO NOTHING;

INSERT INTO chapters (work_id, chapter_number, title, content, word_count, is_draft, published_at)
VALUES
    ('a0000000-0000-0000-0000-000000000001', 1, 'Chapter 1',
     '<p>The library was quiet, the way Hermione liked it.</p>', 9, false, NOW() - INTERVAL '2 days'),
    ('a0000000-0000-0000-0000-000000000002', 1, 'A Study in Fog',
     '<p>London had not seen a fog like it in forty years.</p>', 11, false, NOW() - INTERVAL '1 week')
ON CONFLICT (work_id, chapter_number) DO NOTHING;

INSERT INTO creatorships (creation_id, creation_type, pseud_id)
SELECT w.id, 'Work', p.id
FROM works w JOIN pseuds p ON p.user_id = w.user_id AND p.is_default
WHERE w.id::text LIKE 'a0000000-0000-0000-0000-00000000000_'
  AND NOT EXISTS (SELECT 1 FROM creatorships c WHERE c.creation_id = w.id AND c.creation_type = 'Work');

INSERT INTO work_tags (work_id, tag_id)
SELECT w.id, t.id
FROM works w
JOIN tags t ON t.name = ANY (w.fandoms || w.characters || w.relationships || w.freeform_tags)
WHERE w.id::text LIKE 'a0000000-0000-0000-0000-00000000000_'
ON CONFLICT DO NOTHING;
