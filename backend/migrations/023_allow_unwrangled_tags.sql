-- 002 required every tag to be either canonical or a synonym of a canonical
-- tag (CHECK ((canonical_name IS NULL) = is_canonical)). That made it
-- impossible to create a new tag from the posting form: on AO3 a new tag is
-- neither until a wrangler gets to it.
--
-- What still can't happen: a canonical tag that is also a synonym of another.
ALTER TABLE tags DROP CONSTRAINT IF EXISTS canonical_logic;
ALTER TABLE tags ADD CONSTRAINT canonical_not_synonym
    CHECK (NOT (is_canonical AND canonical_name IS NOT NULL));
