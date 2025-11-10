-- Nuclear AO3: Test Data Seeding
-- This migration adds realistic test data for development and testing
-- Based on actual AO3 usage patterns and popular fandoms

-- =====================================================
-- SAMPLE USERS
-- =====================================================

-- Create some test users with varied activity patterns
INSERT INTO users (id, username, email, display_name, created_at, updated_at) VALUES
    ('11111111-1111-1111-1111-111111111111', 'testwriter01', 'test01@example.com', 'Aspiring Author', NOW() - INTERVAL '6 months', NOW()),
    ('22222222-2222-2222-2222-222222222222', 'fanfic_lover', 'test02@example.com', 'Fic Enthusiast', NOW() - INTERVAL '1 year', NOW()),
    ('33333333-3333-3333-3333-333333333333', 'bookworm2024', 'test03@example.com', 'Book Worm', NOW() - INTERVAL '2 years', NOW()),
    ('44444444-4444-4444-4444-444444444444', 'story_seeker', 'test04@example.com', 'Story Seeker', NOW() - INTERVAL '8 months', NOW()),
    ('55555555-5555-5555-5555-555555555555', 'creative_soul', 'test05@example.com', 'Creative Soul', NOW() - INTERVAL '3 months', NOW())
ON CONFLICT (username) DO NOTHING;

-- =====================================================
-- SAMPLE FANDOMS & TAGS
-- =====================================================

-- Popular fandoms with realistic usage patterns
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
-- SAMPLE WORKS
-- =====================================================

-- Realistic works with varied characteristics
INSERT INTO works (id, title, summary, user_id, language, rating, category, warnings, word_count, chapter_count, expected_chapters, is_complete, is_draft, status, published_at, created_at, updated_at) VALUES
    (
        'w0000000-0000-0000-0000-000000000001',
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
        'w0000000-0000-0000-0000-000000000002',
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
        'w0000000-0000-0000-0000-000000000003',
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
        'w0000000-0000-0000-0000-000000000004',
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
        'w0000000-0000-0000-0000-000000000005',
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
-- WORK STATISTICS
-- =====================================================

-- Add realistic statistics for the sample works
INSERT INTO work_statistics (work_id, hits, kudos, comments, bookmarks, hit_count, kudos_count, comment_count, bookmark_count, created_at, updated_at) VALUES
    ('w0000000-0000-0000-0000-000000000001', 2450, 387, 52, 94, 2450, 387, 52, 94, NOW(), NOW()),
    ('w0000000-0000-0000-0000-000000000002', 4210, 672, 89, 156, 4210, 672, 89, 156, NOW(), NOW()),
    ('w0000000-0000-0000-0000-000000000003', 1560, 234, 31, 67, 1560, 234, 31, 67, NOW(), NOW()),
    ('w0000000-0000-0000-0000-000000000004', 6789, 1023, 145, 298, 6789, 1023, 145, 298, NOW(), NOW()),
    ('w0000000-0000-0000-0000-000000000005', 890, 145, 18, 34, 890, 145, 18, 34, NOW(), NOW())
ON CONFLICT (work_id) DO NOTHING;

-- =====================================================
-- WORK-TAG RELATIONSHIPS
-- =====================================================

-- Tag the Marvel work
INSERT INTO work_tags (work_id, tag_id, created_at) VALUES
    ('w0000000-0000-0000-0000-000000000001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', NOW()), -- Marvel fandom
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', NOW()), -- Tony Stark
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000002', NOW()), -- Steve Rogers
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000101', NOW()), -- Steve/Tony
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000201', NOW()), -- Alternate Universe
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000205', NOW()), -- Friends to Lovers
    ('w0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000207', NOW()), -- Slow Burn

-- Tag the Harry Potter work
    ('w0000000-0000-0000-0000-000000000002', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', NOW()), -- Harry Potter fandom
    ('w0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000003', NOW()), -- Harry Potter
    ('w0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000102', NOW()), -- Harry/Draco
    ('w0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000206', NOW()), -- Enemies to Lovers
    ('w0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000203', NOW()), -- Angst
    ('w0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000204', NOW()), -- Hurt/Comfort

-- Tag the Sherlock work
    ('w0000000-0000-0000-0000-000000000003', 'cccccccc-cccc-cccc-cccc-cccccccccccc', NOW()), -- Sherlock fandom
    ('w0000000-0000-0000-0000-000000000003', 'f0000000-0000-0000-0000-000000000005', NOW()), -- Sherlock Holmes
    ('w0000000-0000-0000-0000-000000000003', 'f0000000-0000-0000-0000-000000000006', NOW()), -- John Watson
    ('w0000000-0000-0000-0000-000000000003', 'f0000000-0000-0000-0000-000000000103', NOW()), -- Sherlock/John
    ('w0000000-0000-0000-0000-000000000003', 'f0000000-0000-0000-0000-000000000202', NOW()), -- Fluff
    ('w0000000-0000-0000-0000-000000000003', 'f0000000-0000-0000-0000-000000000208', NOW()), -- First Kiss

-- Tag the Supernatural work
    ('w0000000-0000-0000-0000-000000000004', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', NOW()), -- Supernatural fandom
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000007', NOW()), -- Dean Winchester
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000009', NOW()), -- Castiel
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000104', NOW()), -- Dean/Castiel
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000203', NOW()), -- Angst
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000204', NOW()), -- Hurt/Comfort
    ('w0000000-0000-0000-0000-000000000004', 'f0000000-0000-0000-0000-000000000205', NOW()), -- Friends to Lovers

-- Tag the coffee shop work  
    ('w0000000-0000-0000-0000-000000000005', 'f0000000-0000-0000-0000-000000000201', NOW()), -- Alternate Universe
    ('w0000000-0000-0000-0000-000000000005', 'f0000000-0000-0000-0000-000000000202', NOW()), -- Fluff
    ('w0000000-0000-0000-0000-000000000005', 'f0000000-0000-0000-0000-000000000209', NOW()) -- Domestic Fluff
ON CONFLICT (work_id, tag_id) DO NOTHING;

-- =====================================================
-- SAMPLE INTERACTIONS
-- =====================================================

-- Add some kudos
INSERT INTO kudos (work_id, user_id, created_at) VALUES
    ('w0000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', NOW() - INTERVAL '1 month'),
    ('w0000000-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', NOW() - INTERVAL '3 weeks'),
    ('w0000000-0000-0000-0000-000000000001', '44444444-4444-4444-4444-444444444444', NOW() - INTERVAL '2 weeks'),
    ('w0000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', NOW() - INTERVAL '2 months'),
    ('w0000000-0000-0000-0000-000000000002', '33333333-3333-3333-3333-333333333333', NOW() - INTERVAL '1 month'),
    ('w0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', NOW() - INTERVAL '3 weeks'),
    ('w0000000-0000-0000-0000-000000000003', '44444444-4444-4444-4444-444444444444', NOW() - INTERVAL '2 weeks')
ON CONFLICT (work_id, user_id) DO NOTHING;

-- Add some comments
INSERT INTO comments (work_id, user_id, content, created_at, updated_at) VALUES
    ('w0000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'This is so sweet! I love the slow development of their relationship.', NOW() - INTERVAL '1 month', NOW() - INTERVAL '1 month'),
    ('w0000000-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', 'The motorcycle repair scenes are so well written!', NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks'),
    ('w0000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'I can''t wait to see how this develops. The tension is perfect!', NOW() - INTERVAL '2 months', NOW() - INTERVAL '2 months'),
    ('w0000000-0000-0000-0000-000000000003', '44444444-4444-4444-4444-444444444444', 'This was exactly what I needed today. Perfect fluff!', NOW() - INTERVAL '2 weeks', NOW() - INTERVAL '2 weeks')
ON CONFLICT DO NOTHING;

-- Add some bookmarks
INSERT INTO bookmarks (work_id, user_id, notes, is_private, is_rec, created_at, updated_at) VALUES
    ('w0000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'Mechanic AU, so good!', false, true, NOW() - INTERVAL '1 month', NOW() - INTERVAL '1 month'),
    ('w0000000-0000-0000-0000-000000000002', '33333333-3333-3333-3333-333333333333', 'Eighth year fic, enemies to lovers', false, false, NOW() - INTERVAL '3 months', NOW() - INTERVAL '3 months'),
    ('w0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'Perfect Johnlock fluff', false, true, NOW() - INTERVAL '3 weeks', NOW() - INTERVAL '3 weeks'),
    ('w0000000-0000-0000-0000-000000000004', '55555555-5555-5555-5555-555555555555', 'Amazing Destiel longfic', false, true, NOW() - INTERVAL '5 months', NOW() - INTERVAL '2 weeks')
ON CONFLICT (work_id, user_id) DO NOTHING;

-- =====================================================
-- UPDATE STATISTICS
-- =====================================================

-- Update work statistics to match the interactions we just added
UPDATE works SET 
    kudos_count = (SELECT COUNT(*) FROM kudos WHERE work_id = works.id),
    comment_count = (SELECT COUNT(*) FROM comments WHERE work_id = works.id AND is_deleted = false),
    bookmark_count = (SELECT COUNT(*) FROM bookmarks WHERE work_id = works.id),
    updated_at = NOW()
WHERE id IN (
    'w0000000-0000-0000-0000-000000000001',
    'w0000000-0000-0000-0000-000000000002', 
    'w0000000-0000-0000-0000-000000000003',
    'w0000000-0000-0000-0000-000000000004',
    'w0000000-0000-0000-0000-000000000005'
);

-- Update tag use counts
UPDATE tags SET use_count = (
    SELECT COUNT(*) FROM work_tags wt 
    JOIN works w ON wt.work_id = w.id 
    WHERE wt.tag_id = tags.id AND w.is_draft = false
) WHERE id IN (
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
    'cccccccc-cccc-cccc-cccc-cccccccccccc',
    'dddddddd-dddd-dddd-dddd-dddddddddddd',
    'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee'
);

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE users IS 'Sample test users for development and demonstration';
COMMENT ON TABLE works IS 'Sample works covering popular fandoms and diverse content types';
COMMENT ON TABLE work_statistics IS 'Realistic interaction statistics matching actual AO3 patterns';

-- Success message
DO $$
BEGIN
    RAISE NOTICE '✅ Test data seeded successfully!';
    RAISE NOTICE '   👥 5 test users created';
    RAISE NOTICE '   🏷️  25 tags created (fandoms, characters, relationships, freeform)';
    RAISE NOTICE '   📚 5 sample works created';
    RAISE NOTICE '   💬 Interactions added (kudos, comments, bookmarks)';
    RAISE NOTICE '   🔗 Work-tag relationships established';
    RAISE NOTICE '';
    RAISE NOTICE '🔍 You can now test the APIs with realistic data!';
END $$;