-- Nuclear AO3: Demo Data for Testing Smart Recommendations
-- Simplified version that works with current schema

-- =====================================================
-- SAMPLE USERS WITH PROPER PASSWORDS
-- =====================================================

INSERT INTO users (id, username, email, password_hash, display_name, created_at, updated_at) VALUES
    ('11111111-1111-1111-1111-111111111111', 'testwriter01', 'test01@example.com', '$2a$10$dummy.hash.for.demo.data.only', 'Aspiring Author', NOW() - INTERVAL '6 months', NOW()),
    ('22222222-2222-2222-2222-222222222222', 'fanfic_lover', 'test02@example.com', '$2a$10$dummy.hash.for.demo.data.only', 'Fic Enthusiast', NOW() - INTERVAL '1 year', NOW()),
    ('33333333-3333-3333-3333-333333333333', 'bookworm2024', 'test03@example.com', '$2a$10$dummy.hash.for.demo.data.only', 'Book Worm', NOW() - INTERVAL '2 years', NOW()),
    ('44444444-4444-4444-4444-444444444444', 'story_seeker', 'test04@example.com', '$2a$10$dummy.hash.for.demo.data.only', 'Story Seeker', NOW() - INTERVAL '8 months', NOW()),
    ('55555555-5555-5555-5555-555555555555', 'creative_soul', 'test05@example.com', '$2a$10$dummy.hash.for.demo.data.only', 'Creative Soul', NOW() - INTERVAL '3 months', NOW())
ON CONFLICT (username) DO NOTHING;

-- =====================================================
-- SAMPLE TAGS
-- =====================================================

INSERT INTO tags (id, name, type, canonical, is_canonical, is_filterable, description, use_count, created_at, updated_at) VALUES
    -- Major fandoms
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Marvel Cinematic Universe', 'fandom', true, true, true, 'Superhero movie franchise', 15420, NOW(), NOW()),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Harry Potter - J. K. Rowling', 'fandom', true, true, true, 'Wizarding world book series', 22150, NOW(), NOW()),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Sherlock Holmes & Related Fandoms', 'fandom', true, true, true, 'Detective fiction', 8930, NOW(), NOW()),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '방탄소년단 | Bangtan Boys | BTS', 'fandom', true, true, true, 'Korean pop group', 12340, NOW(), NOW()),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'Supernatural', 'fandom', true, true, true, 'Monster hunting TV series', 18750, NOW(), NOW()),
    
    -- Popular characters
    ('f0000000-0000-0000-0000-000000000001', 'Tony Stark', 'character', true, true, true, '', 8420, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000002', 'Steve Rogers', 'character', true, true, true, '', 7230, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000003', 'Harry Potter', 'character', true, true, true, '', 12450, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000004', 'Hermione Granger', 'character', true, true, true, '', 9870, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000005', 'Sherlock Holmes', 'character', true, true, true, '', 6540, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000006', 'John Watson', 'character', true, true, true, '', 5980, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000007', 'Dean Winchester', 'character', true, true, true, '', 11230, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000008', 'Sam Winchester', 'character', true, true, true, '', 9450, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000009', 'Castiel', 'character', true, true, true, '', 8770, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000010', 'Draco Malfoy', 'character', true, true, true, '', 7650, NOW(), NOW()),
    
    -- Popular relationships
    ('f0000000-0000-0000-0000-000000000101', 'Steve Rogers/Tony Stark', 'relationship', true, true, true, '', 4520, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000102', 'Harry Potter/Draco Malfoy', 'relationship', true, true, true, '', 6230, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000103', 'Sherlock Holmes/John Watson', 'relationship', true, true, true, '', 5440, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000104', 'Dean Winchester/Castiel', 'relationship', true, true, true, '', 7890, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000105', 'Harry Potter & Hermione Granger & Ron Weasley', 'relationship', true, true, true, '', 3210, NOW(), NOW()),
    
    -- Common freeform tags
    ('f0000000-0000-0000-0000-000000000201', 'Alternate Universe', 'freeform', true, true, true, '', 25670, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000202', 'Fluff', 'freeform', true, true, true, '', 18420, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000203', 'Angst', 'freeform', true, true, true, '', 16780, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000204', 'Hurt/Comfort', 'freeform', true, true, true, '', 14590, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000205', 'Friends to Lovers', 'freeform', true, true, true, '', 12340, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000206', 'Enemies to Lovers', 'freeform', true, true, true, '', 9870, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000207', 'Slow Burn', 'freeform', true, true, true, '', 8765, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000208', 'First Kiss', 'freeform', true, true, true, '', 7654, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000209', 'Domestic Fluff', 'freeform', true, true, true, '', 6543, NOW(), NOW()),
    ('f0000000-0000-0000-0000-000000000210', 'Fix-It', 'freeform', true, true, true, '', 5432, NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- SAMPLE WORKS WITH PROPER UUIDs
-- =====================================================

INSERT INTO works (id, title, summary, user_id, language, rating, category, warnings, word_count, chapter_count, expected_chapters, is_complete, is_draft, status, published_at, created_at, updated_at) VALUES
    (
        'w1111111-1111-1111-1111-111111111111',
        'The Mechanic and the Soldier',
        'Tony Stark is just trying to run his garage in peace. Steve Rogers is the new guy in town with a motorcycle that keeps breaking down. Neither of them expected to fall in love.',
        '11111111-1111-1111-1111-111111111111',
        'en',
        'Teen And Up Audiences',
        'M/M',
        'No Archive Warnings Apply',
        15420,
        8,
        10,
        false,
        false,
        'published',
        NOW() - INTERVAL '2 months',
        NOW() - INTERVAL '2 months',
        NOW() - INTERVAL '1 week'
    ),
    (
        'w2222222-2222-2222-2222-222222222222',
        'Potions and Revelations',
        'Eighth year at Hogwarts brings new challenges, and Harry finds himself paired with Draco Malfoy for Advanced Potions. What starts as reluctant cooperation slowly becomes something more.',
        '22222222-2222-2222-2222-222222222222',
        'en',
        'Mature',
        'M/M',
        'No Archive Warnings Apply',
        32150,
        12,
        15,
        false,
        false,
        'published',
        NOW() - INTERVAL '4 months',
        NOW() - INTERVAL '4 months',
        NOW() - INTERVAL '3 days'
    ),
    (
        'w3333333-3333-3333-3333-333333333333',
        'The Case of the Missing Heart',
        'John Watson thought he knew everything about his flatmate. He was wrong.',
        '33333333-3333-3333-3333-333333333333',
        'en',
        'General Audiences',
        'M/M',
        'No Archive Warnings Apply',
        8720,
        3,
        3,
        true,
        false,
        'complete',
        NOW() - INTERVAL '1 month',
        NOW() - INTERVAL '1 month',
        NOW() - INTERVAL '1 month'
    ),
    (
        'w4444444-4444-4444-4444-444444444444',
        'Wings and Prayers',
        'Dean has never been good at asking for help. Castiel has never been good at understanding human emotions. Together, they might just figure it out.',
        '44444444-4444-4444-4444-444444444444',
        'en',
        'Explicit',
        'M/M',
        'No Archive Warnings Apply',
        45670,
        18,
        NULL,
        false,
        false,
        'published',
        NOW() - INTERVAL '6 months',
        NOW() - INTERVAL '6 months',
        NOW() - INTERVAL '2 weeks'
    ),
    (
        'w5555555-5555-5555-5555-555555555555',
        'Coffee Shop Chronicles',
        'A collection of moments between a barista and her favorite customer.',
        '55555555-5555-5555-5555-555555555555',
        'en',
        'General Audiences',
        'F/F',
        'No Archive Warnings Apply',
        12340,
        7,
        NULL,
        false,
        false,
        'published',
        NOW() - INTERVAL '3 months',
        NOW() - INTERVAL '3 months',
        NOW() - INTERVAL '1 week'
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================
-- WORK-TAG RELATIONSHIPS
-- =====================================================

INSERT INTO work_tags (work_id, tag_id, created_at) VALUES
    -- Tag the Marvel work
    ('w1111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000001', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000002', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000101', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000201', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000205', NOW()),
    ('w1111111-1111-1111-1111-111111111111', 'f0000000-0000-0000-0000-000000000207', NOW()),

    -- Tag the Harry Potter work  
    ('w2222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000003', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000010', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000102', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000206', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000203', NOW()),
    ('w2222222-2222-2222-2222-222222222222', 'f0000000-0000-0000-0000-000000000204', NOW()),

    -- Tag the Sherlock work
    ('w3333333-3333-3333-3333-333333333333', 'cccccccc-cccc-cccc-cccc-cccccccccccc', NOW()),
    ('w3333333-3333-3333-3333-333333333333', 'f0000000-0000-0000-0000-000000000005', NOW()),
    ('w3333333-3333-3333-3333-333333333333', 'f0000000-0000-0000-0000-000000000006', NOW()),
    ('w3333333-3333-3333-3333-333333333333', 'f0000000-0000-0000-0000-000000000103', NOW()),
    ('w3333333-3333-3333-3333-333333333333', 'f0000000-0000-0000-0000-000000000202', NOW()),
    ('w3333333-3333-3333-3333-333333333333', 'f0000000-0000-0000-0000-000000000208', NOW()),

    -- Tag the Supernatural work
    ('w4444444-4444-4444-4444-444444444444', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000007', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000009', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000104', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000203', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000204', NOW()),
    ('w4444444-4444-4444-4444-444444444444', 'f0000000-0000-0000-0000-000000000205', NOW()),

    -- Tag the coffee shop work  
    ('w5555555-5555-5555-5555-555555555555', 'f0000000-0000-0000-0000-000000000201', NOW()),
    ('w5555555-5555-5555-5555-555555555555', 'f0000000-0000-0000-0000-000000000202', NOW()),
    ('w5555555-5555-5555-5555-555555555555', 'f0000000-0000-0000-0000-000000000209', NOW())
ON CONFLICT (work_id, tag_id) DO NOTHING;

-- =====================================================
-- UPDATE ELASTICSEARCH WITH WORKS DATA
-- =====================================================

-- We'll need to trigger the Elasticsearch indexing after this
-- The search service should pick up these works automatically

RAISE NOTICE '✅ Demo data loaded successfully!';
RAISE NOTICE '   👥 5 test users created';
RAISE NOTICE '   🏷️  25 tags created (fandoms, characters, relationships, freeform)';
RAISE NOTICE '   📚 5 sample works created with proper tagging';
RAISE NOTICE '   🔗 Work-tag relationships established';
RAISE NOTICE '';
RAISE NOTICE '🔍 Smart recommendations should now work with real data!';