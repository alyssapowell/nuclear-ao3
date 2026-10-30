-- Work search runs in Postgres instead of Elasticsearch.
--
-- Each work gets a search_vector column that Postgres keeps up to date by
-- itself whenever the title, summary or tags change, so there's no separate
-- index to rebuild or get out of sync. Matches in the title count most, then
-- tags, then the summary. Chapter text isn't searched (AO3 doesn't either).

-- Postgres only allows IMMUTABLE functions in a generated column.
-- array_to_string is marked STABLE because some element types print
-- differently depending on settings; for text it's always the same, so this
-- wrapper is safe to mark IMMUTABLE.
CREATE FUNCTION work_search_vector(
    title TEXT, summary TEXT,
    fandoms TEXT[], characters TEXT[], relationships TEXT[], freeform_tags TEXT[]
) RETURNS tsvector LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT setweight(to_tsvector('english', coalesce(title, '')), 'A')
        || setweight(to_tsvector('english', coalesce(array_to_string(
               coalesce(fandoms, '{}') || coalesce(characters, '{}')
               || coalesce(relationships, '{}') || coalesce(freeform_tags, '{}'), ' '), '')), 'B')
        || setweight(to_tsvector('english', coalesce(summary, '')), 'C')
$$;

ALTER TABLE works ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (work_search_vector(title, summary, fandoms, characters, relationships, freeform_tags)) STORED;

CREATE INDEX idx_works_search_vector ON works USING gin (search_vector);

-- Tag filters ("works in this fandom", "without this tag").
CREATE INDEX idx_works_fandoms ON works USING gin (fandoms);
CREATE INDEX idx_works_characters ON works USING gin (characters);
CREATE INDEX idx_works_relationships ON works USING gin (relationships);
CREATE INDEX idx_works_freeform_tags ON works USING gin (freeform_tags);

-- Replaced by search_vector. Indexing every chapter's full text was the
-- largest index in the database and nothing searched it.
DROP INDEX IF EXISTS idx_works_text_search;
DROP INDEX IF EXISTS idx_chapters_content_search;
