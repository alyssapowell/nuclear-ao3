-- Tag search and pseud ("people") search match part of a name anywhere in it
-- (LIKE '%part%'), which a normal index can't serve. Trigram indexes do.
--
-- pg_trgm ships with Postgres and is allowed for ordinary database owners
-- since Postgres 13. If it can't be installed here, the searches still work;
-- they just read the whole table, which is fine for a small archive.
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
    CREATE INDEX IF NOT EXISTS idx_tags_name_trgm ON tags USING gin (lower(name::text) gin_trgm_ops);
    CREATE INDEX IF NOT EXISTS idx_pseuds_name_trgm ON pseuds USING gin (lower(name::text) gin_trgm_ops);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_trgm is not available (%); name searches will not use an index', SQLERRM;
END
$$;
