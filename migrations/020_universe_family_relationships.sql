-- Universe Family Relationships Migration
-- This creates hierarchical relationships between fandom tags to enable crossover detection

-- Create universe family parent tags if they don't exist
INSERT INTO tags (id, name, type, is_canonical, description)
VALUES 
  ('11111111-1111-1111-1111-111111111111', 'Marvel Universe Family', 'additional', true, 'Parent tag for all Marvel-related fandoms'),
  ('22222222-2222-2222-2222-222222222222', 'Harry Potter Universe Family', 'additional', true, 'Parent tag for all Harry Potter-related fandoms'),
  ('33333333-3333-3333-3333-333333333333', 'DC Universe Family', 'additional', true, 'Parent tag for all DC Comics-related fandoms'),
  ('44444444-4444-4444-4444-444444444444', 'Star Wars Universe Family', 'additional', true, 'Parent tag for all Star Wars-related fandoms'),
  ('55555555-5555-5555-5555-555555555555', 'Sherlock Holmes Universe Family', 'additional', true, 'Parent tag for all Sherlock Holmes-related fandoms')
ON CONFLICT (name) DO NOTHING;

-- Create relationships for Marvel Universe Family
WITH marvel_parent AS (
  SELECT id FROM tags WHERE name = 'Marvel Universe Family'
), marvel_tags AS (
  SELECT id FROM tags 
  WHERE type = 'fandom' 
  AND (
    name ILIKE '%marvel%' 
    OR name ILIKE '%avengers%' 
    OR name ILIKE '%iron man%'
    OR name ILIKE '%captain america%'
    OR name ILIKE '%thor%'
    OR name ILIKE '%spider-man%'
    OR name ILIKE '%x-men%'
    OR name ILIKE '%deadpool%'
    OR name ILIKE '%guardians of the galaxy%'
  )
)
INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_by)
SELECT 
  mp.id,
  mt.id,
  'parent_child',
  '123e4567-e89b-12d3-a456-426614174003'::uuid
FROM marvel_parent mp, marvel_tags mt
ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING;

-- Create relationships for Harry Potter Universe Family
WITH hp_parent AS (
  SELECT id FROM tags WHERE name = 'Harry Potter Universe Family'
), hp_tags AS (
  SELECT id FROM tags 
  WHERE type = 'fandom' 
  AND (
    name ILIKE '%harry potter%' 
    OR name ILIKE '%fantastic beasts%'
    OR name ILIKE '%hogwarts%'
    OR name ILIKE '%cursed child%'
  )
)
INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_by)
SELECT 
  hp.id,
  ht.id,
  'parent_child',
  '123e4567-e89b-12d3-a456-426614174003'::uuid
FROM hp_parent hp, hp_tags ht
ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING;

-- Create relationships for DC Universe Family
WITH dc_parent AS (
  SELECT id FROM tags WHERE name = 'DC Universe Family'
), dc_tags AS (
  SELECT id FROM tags 
  WHERE type = 'fandom' 
  AND (
    name ILIKE '%batman%' 
    OR name ILIKE '%superman%'
    OR name ILIKE '%wonder woman%'
    OR name ILIKE '%justice league%'
    OR name ILIKE '%dc%'
    OR name ILIKE '%flash%'
    OR name ILIKE '%green lantern%'
    OR name ILIKE '%aquaman%'
  )
)
INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_by)
SELECT 
  dp.id,
  dt.id,
  'parent_child',
  '123e4567-e89b-12d3-a456-426614174003'::uuid
FROM dc_parent dp, dc_tags dt
ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING;

-- Create relationships for Star Wars Universe Family
WITH sw_parent AS (
  SELECT id FROM tags WHERE name = 'Star Wars Universe Family'
), sw_tags AS (
  SELECT id FROM tags 
  WHERE type = 'fandom' 
  AND (
    name ILIKE '%star wars%'
    OR name ILIKE '%jedi%'
    OR name ILIKE '%sith%'
    OR name ILIKE '%mandalorian%'
  )
)
INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_by)
SELECT 
  sp.id,
  st.id,
  'parent_child',
  '123e4567-e89b-12d3-a456-426614174003'::uuid
FROM sw_parent sp, sw_tags st
ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING;

-- Create relationships for Sherlock Holmes Universe Family
WITH sherlock_parent AS (
  SELECT id FROM tags WHERE name = 'Sherlock Holmes Universe Family'
), sherlock_tags AS (
  SELECT id FROM tags 
  WHERE type = 'fandom' 
  AND (
    name ILIKE '%sherlock%'
    OR name ILIKE '%holmes%'
    OR name ILIKE '%watson%'
    OR name ILIKE '%baker street%'
  )
)
INSERT INTO tag_relationships (parent_tag_id, child_tag_id, relationship_type, created_by)
SELECT 
  sp.id,
  st.id,
  'parent_child',
  '123e4567-e89b-12d3-a456-426614174003'::uuid
FROM sherlock_parent sp, sherlock_tags st
ON CONFLICT (parent_tag_id, child_tag_id) DO NOTHING;

-- Create indexes for efficient crossover detection queries
CREATE INDEX IF NOT EXISTS idx_tag_relationships_universe_families 
ON tag_relationships (parent_tag_id) 
WHERE parent_tag_id IN (
  SELECT id FROM tags WHERE name LIKE '%Universe Family'
);

-- Verify the relationships were created
SELECT 
  p.name as universe_family,
  COUNT(tr.child_tag_id) as member_count
FROM tags p
JOIN tag_relationships tr ON p.id = tr.parent_tag_id
WHERE p.type = 'meta' AND p.name LIKE '%Universe Family'
GROUP BY p.id, p.name
ORDER BY p.name;