-- Nuclear AO3: Complete Content Management Schema  
-- This migration creates all tables for works, chapters, tags, and relationships
-- Designed for high performance with millions of works and proper search indexing

-- =====================================================
-- CONTENT TABLES
-- =====================================================

-- Works table - core fanfiction content
CREATE TABLE works (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    summary TEXT,
    notes TEXT, -- Author's notes at the beginning
    end_notes TEXT, -- Author's notes at the end
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    
    -- Content ratings and warnings  
    rating VARCHAR(50) NOT NULL DEFAULT 'Not Rated',
    archive_warning VARCHAR(100),
    category VARCHAR(100), -- Gen, F/M, M/M, F/F, Multi, Other
    
    -- Work statistics
    word_count INTEGER NOT NULL DEFAULT 0,
    chapter_count INTEGER NOT NULL DEFAULT 1,
    expected_chapters INTEGER, -- NULL for unknown, number for planned
    hit_count INTEGER NOT NULL DEFAULT 0,
    kudos_count INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    bookmark_count INTEGER NOT NULL DEFAULT 0,
    
    -- Publishing info
    is_complete BOOLEAN DEFAULT false,
    is_draft BOOLEAN DEFAULT true,
    published_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    imported_from VARCHAR(100), -- If imported from another site
    external_id VARCHAR(255), -- Original ID if imported
    restricted BOOLEAN DEFAULT false, -- Requires login to view
    moderated BOOLEAN DEFAULT false, -- Flagged for moderation
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rating_values CHECK (rating IN ('Not Rated', 'General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit')),
    CONSTRAINT word_count_positive CHECK (word_count >= 0),
    CONSTRAINT chapter_counts_valid CHECK (chapter_count > 0 AND (expected_chapters IS NULL OR expected_chapters >= chapter_count))
);

-- Indexes for works
CREATE INDEX idx_works_user_id ON works(user_id);
CREATE INDEX idx_works_published_at ON works(published_at) WHERE published_at IS NOT NULL;
CREATE INDEX idx_works_updated_at ON works(updated_at);
CREATE INDEX idx_works_rating ON works(rating);
CREATE INDEX idx_works_language ON works(language);
CREATE INDEX idx_works_word_count ON works(word_count);
CREATE INDEX idx_works_is_complete ON works(is_complete);
CREATE INDEX idx_works_is_draft ON works(is_draft);
CREATE INDEX idx_works_category ON works(category);
CREATE INDEX idx_works_kudos_count ON works(kudos_count);
CREATE INDEX idx_works_hit_count ON works(hit_count);

-- Composite indexes for common queries
CREATE INDEX idx_works_published_updated ON works(published_at DESC, updated_at DESC) WHERE is_draft = false;
CREATE INDEX idx_works_user_published ON works(user_id, published_at DESC) WHERE is_draft = false;
CREATE INDEX idx_works_rating_language ON works(rating, language) WHERE is_draft = false;
CREATE INDEX idx_works_complete_updated ON works(is_complete, updated_at DESC) WHERE is_draft = false;

-- Full text search index
CREATE INDEX idx_works_text_search ON works 
    USING gin(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(summary, '') || ' ' || coalesce(notes, '')));

-- Chapters table
CREATE TABLE chapters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    title VARCHAR(500),
    summary TEXT,
    notes TEXT, -- Chapter-specific notes
    end_notes TEXT,
    content TEXT NOT NULL,
    chapter_number INTEGER NOT NULL,
    word_count INTEGER NOT NULL DEFAULT 0,
    is_draft BOOLEAN DEFAULT false,
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(work_id, chapter_number),
    CONSTRAINT chapter_number_positive CHECK (chapter_number > 0),
    CONSTRAINT word_count_positive CHECK (word_count >= 0)
);

-- Indexes for chapters
CREATE INDEX idx_chapters_work_id ON chapters(work_id, chapter_number);
CREATE INDEX idx_chapters_published_at ON chapters(published_at) WHERE published_at IS NOT NULL;
CREATE INDEX idx_chapters_word_count ON chapters(word_count);

-- Full text search on chapter content
CREATE INDEX idx_chapters_content_search ON chapters 
    USING gin(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, '')));

-- Tags table - hierarchical tag system
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name CITEXT UNIQUE NOT NULL,
    canonical_name CITEXT, -- For synonyms, points to canonical form
    type VARCHAR(20) NOT NULL DEFAULT 'freeform',
    description TEXT,
    is_canonical BOOLEAN DEFAULT false,
    is_filterable BOOLEAN DEFAULT true,
    use_count INTEGER DEFAULT 0, -- How many works use this tag
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT tag_type_values CHECK (type IN ('fandom', 'character', 'relationship', 'freeform', 'rating', 'warning', 'category', 'additional')),
    CONSTRAINT canonical_logic CHECK ((canonical_name IS NULL) = is_canonical)
);

-- Indexes for tags
CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_tags_type ON tags(type);
CREATE INDEX idx_tags_canonical_name ON tags(canonical_name) WHERE canonical_name IS NOT NULL;
CREATE INDEX idx_tags_is_canonical ON tags(is_canonical);
CREATE INDEX idx_tags_use_count ON tags(use_count DESC);
CREATE INDEX idx_tags_filterable ON tags(is_filterable) WHERE is_filterable = true;

-- Full text search on tag names and descriptions
CREATE INDEX idx_tags_text_search ON tags 
    USING gin(to_tsvector('english', coalesce(name::text, '') || ' ' || coalesce(description, '')));

-- Junction table for work-tag relationships
CREATE TABLE work_tags (
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (work_id, tag_id)
);

CREATE INDEX idx_work_tags_work_id ON work_tags(work_id);
CREATE INDEX idx_work_tags_tag_id ON work_tags(tag_id);

-- Tag relationships (for tag wrangling)
CREATE TABLE tag_relationships (
    parent_tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    child_tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    relationship_type VARCHAR(20) NOT NULL DEFAULT 'parent_child',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    PRIMARY KEY (parent_tag_id, child_tag_id),
    CONSTRAINT no_self_reference CHECK (parent_tag_id != child_tag_id),
    CONSTRAINT relationship_types CHECK (relationship_type IN ('parent_child', 'synonym', 'related'))
);

CREATE INDEX idx_tag_relationships_parent ON tag_relationships(parent_tag_id);
CREATE INDEX idx_tag_relationships_child ON tag_relationships(child_tag_id);

-- Series table
CREATE TABLE series (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    notes TEXT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_complete BOOLEAN DEFAULT false,
    restricted BOOLEAN DEFAULT false,
    work_count INTEGER DEFAULT 0,
    word_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_series_user_id ON series(user_id);
CREATE INDEX idx_series_updated_at ON series(updated_at);
CREATE INDEX idx_series_work_count ON series(work_count);

-- Series-work relationships
CREATE TABLE series_works (
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (series_id, work_id),
    UNIQUE(series_id, position)
);

CREATE INDEX idx_series_works_series ON series_works(series_id, position);
CREATE INDEX idx_series_works_work ON series_works(work_id);

-- Comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID REFERENCES works(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapters(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN DEFAULT false,
    is_moderated BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT comment_target CHECK (
        (work_id IS NOT NULL AND chapter_id IS NULL) OR 
        (work_id IS NULL AND chapter_id IS NOT NULL)
    )
);

CREATE INDEX idx_comments_work_id ON comments(work_id, created_at) WHERE work_id IS NOT NULL;
CREATE INDEX idx_comments_chapter_id ON comments(chapter_id, created_at) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent ON comments(parent_comment_id);

-- Kudos table (likes/appreciation)
CREATE TABLE kudos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    guest_session VARCHAR(255), -- For anonymous kudos
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT kudos_identity CHECK (user_id IS NOT NULL OR guest_session IS NOT NULL),
    UNIQUE(work_id, user_id) WHERE user_id IS NOT NULL,
    UNIQUE(work_id, guest_session) WHERE guest_session IS NOT NULL
);

CREATE INDEX idx_kudos_work_id ON kudos(work_id);
CREATE INDEX idx_kudos_user_id ON kudos(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_kudos_created_at ON kudos(created_at);

-- Bookmarks table
CREATE TABLE bookmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notes TEXT,
    tags TEXT[], -- User's personal tags for the bookmark
    is_private BOOLEAN DEFAULT false,
    is_rec BOOLEAN DEFAULT false, -- Recommendation
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(work_id, user_id)
);

CREATE INDEX idx_bookmarks_work_id ON bookmarks(work_id);
CREATE INDEX idx_bookmarks_user_id ON bookmarks(user_id, created_at);
CREATE INDEX idx_bookmarks_private ON bookmarks(is_private);
CREATE INDEX idx_bookmarks_rec ON bookmarks(is_rec) WHERE is_rec = true;

-- Collections table (themed collections, challenges, etc.)
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) DEFAULT 'collection',
    is_open BOOLEAN DEFAULT true,
    is_moderated BOOLEAN DEFAULT false,
    is_anonymous BOOLEAN DEFAULT false,
    work_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT collection_types CHECK (type IN ('collection', 'challenge', 'exchange', 'prompt_meme'))
);

CREATE INDEX idx_collections_user_id ON collections(user_id);
CREATE INDEX idx_collections_type ON collections(type);
CREATE INDEX idx_collections_is_open ON collections(is_open);
CREATE INDEX idx_collections_updated_at ON collections(updated_at);

-- Collection-work relationships
CREATE TABLE collection_works (
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    work_id UUID NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    approved BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (collection_id, work_id)
);

CREATE INDEX idx_collection_works_collection ON collection_works(collection_id, approved);
CREATE INDEX idx_collection_works_work ON collection_works(work_id);

-- =====================================================
-- TRIGGERS AND FUNCTIONS FOR STATISTICS
-- =====================================================

-- Function to update work statistics
CREATE OR REPLACE FUNCTION update_work_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'chapters' THEN
        -- Update work word count and chapter count
        UPDATE works SET 
            word_count = (SELECT COALESCE(SUM(word_count), 0) FROM chapters WHERE work_id = NEW.work_id AND is_draft = false),
            chapter_count = (SELECT COUNT(*) FROM chapters WHERE work_id = NEW.work_id AND is_draft = false),
            updated_at = NOW()
        WHERE id = NEW.work_id;
    ELSIF TG_TABLE_NAME = 'kudos' THEN
        -- Update kudos count
        UPDATE works SET 
            kudos_count = (SELECT COUNT(*) FROM kudos WHERE work_id = NEW.work_id),
            updated_at = NOW()
        WHERE id = NEW.work_id;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        -- Update comment count
        UPDATE works SET 
            comment_count = (SELECT COUNT(*) FROM comments WHERE work_id = NEW.work_id AND is_deleted = false),
            updated_at = NOW()
        WHERE id = NEW.work_id;
    ELSIF TG_TABLE_NAME = 'bookmarks' THEN
        -- Update bookmark count
        UPDATE works SET 
            bookmark_count = (SELECT COUNT(*) FROM bookmarks WHERE work_id = NEW.work_id),
            updated_at = NOW()
        WHERE id = NEW.work_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Triggers for statistics
CREATE TRIGGER update_work_stats_chapters
    AFTER INSERT OR UPDATE OR DELETE ON chapters
    FOR EACH ROW EXECUTE FUNCTION update_work_stats();

CREATE TRIGGER update_work_stats_kudos
    AFTER INSERT OR DELETE ON kudos
    FOR EACH ROW EXECUTE FUNCTION update_work_stats();

CREATE TRIGGER update_work_stats_comments
    AFTER INSERT OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_work_stats();

CREATE TRIGGER update_work_stats_bookmarks
    AFTER INSERT OR DELETE ON bookmarks
    FOR EACH ROW EXECUTE FUNCTION update_work_stats();

-- Function to update tag use counts
CREATE OR REPLACE FUNCTION update_tag_use_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags SET use_count = use_count + 1 WHERE id = NEW.tag_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags SET use_count = use_count - 1 WHERE id = OLD.tag_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Trigger for tag use counts
CREATE TRIGGER update_tag_use_count_trigger
    AFTER INSERT OR DELETE ON work_tags
    FOR EACH ROW EXECUTE FUNCTION update_tag_use_count();

-- Function to update series statistics
CREATE OR REPLACE FUNCTION update_series_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'series_works' THEN
        UPDATE series SET
            work_count = (SELECT COUNT(*) FROM series_works WHERE series_id = COALESCE(NEW.series_id, OLD.series_id)),
            word_count = (
                SELECT COALESCE(SUM(w.word_count), 0) 
                FROM works w 
                JOIN series_works sw ON w.id = sw.work_id 
                WHERE sw.series_id = COALESCE(NEW.series_id, OLD.series_id) 
                AND w.is_draft = false
            ),
            updated_at = NOW()
        WHERE id = COALESCE(NEW.series_id, OLD.series_id);
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Trigger for series statistics
CREATE TRIGGER update_series_stats_trigger
    AFTER INSERT OR DELETE ON series_works
    FOR EACH ROW EXECUTE FUNCTION update_series_stats();

-- =====================================================
-- VIEWS FOR COMMON QUERIES
-- =====================================================

-- Published works with tag information
CREATE VIEW published_works_with_stats AS
SELECT 
    w.*,
    u.username as author_username,
    u.display_name as author_display_name,
    COUNT(DISTINCT wt.tag_id) as tag_count,
    ARRAY_AGG(DISTINCT t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL) as tag_names,
    ARRAY_AGG(DISTINCT t.type ORDER BY t.type) FILTER (WHERE t.type IS NOT NULL) as tag_types
FROM works w
JOIN users u ON w.user_id = u.id
LEFT JOIN work_tags wt ON w.id = wt.work_id
LEFT JOIN tags t ON wt.tag_id = t.id
WHERE w.is_draft = false AND w.published_at IS NOT NULL
GROUP BY w.id, u.username, u.display_name;

-- Popular works view (for recommendations)
CREATE VIEW popular_works AS
SELECT 
    w.*,
    u.username as author_username,
    u.display_name as author_display_name,
    (w.kudos_count * 2 + w.comment_count * 3 + w.bookmark_count * 4 + w.hit_count * 0.1) as popularity_score
FROM works w
JOIN users u ON w.user_id = u.id
WHERE w.is_draft = false AND w.published_at IS NOT NULL
ORDER BY popularity_score DESC;

-- Tag statistics view
CREATE VIEW tag_stats AS
SELECT 
    t.*,
    COUNT(DISTINCT wt.work_id) as actual_use_count,
    COUNT(DISTINCT wt.work_id) FILTER (WHERE w.published_at > NOW() - INTERVAL '30 days') as recent_use_count
FROM tags t
LEFT JOIN work_tags wt ON t.id = wt.tag_id
LEFT JOIN works w ON wt.work_id = w.id AND w.is_draft = false
GROUP BY t.id;

-- =====================================================
-- SAMPLE DATA
-- =====================================================

-- Insert sample tags
INSERT INTO tags (name, type, is_canonical) VALUES
    ('Harry Potter - J. K. Rowling', 'fandom', true),
    ('Sherlock Holmes & Related Fandoms', 'fandom', true),
    ('Marvel Cinematic Universe', 'fandom', true),
    ('Hermione Granger', 'character', true),
    ('Harry Potter', 'character', true),
    ('Sherlock Holmes', 'character', true),
    ('John Watson', 'character', true),
    ('Harry Potter/Hermione Granger', 'relationship', true),
    ('Sherlock Holmes/John Watson', 'relationship', true),
    ('Alternate Universe', 'freeform', true),
    ('Fluff', 'freeform', true),
    ('Angst', 'freeform', true),
    ('Hurt/Comfort', 'freeform', true),
    ('Friends to Lovers', 'freeform', true),
    ('Modern AU', 'freeform', true)
ON CONFLICT (name) DO NOTHING;

-- Insert sample works
INSERT INTO works (id, title, summary, user_id, rating, category, word_count, chapter_count, is_draft, published_at) VALUES
    (
        uuid_generate_v4(),
        'The Brightest Witch of Her Age',
        'Hermione Granger has always been the smartest witch in her year, but what happens when she discovers magic that goes beyond anything taught at Hogwarts?',
        '123e4567-e89b-12d3-a456-426614174000',
        'Teen And Up Audiences',
        'F/M',
        15420,
        5,
        false,
        NOW() - INTERVAL '2 days'
    ),
    (
        uuid_generate_v4(),
        'Baker Street Mysteries',
        'A collection of cases that never made it into Dr. Watson''s published accounts. Some things are too dangerous for the public to know.',
        '123e4567-e89b-12d3-a456-426614174001',
        'Mature',
        'M/M',
        32150,
        12,
        false,
        NOW() - INTERVAL '1 week'
    ),
    (
        uuid_generate_v4(),
        'Coffee Shop Chronicles', 
        'Tony Stark owns a coffee shop. Steve Rogers is a regular customer. Love brews slowly.',
        '123e4567-e89b-12d3-a456-426614174000',
        'General Audiences',
        'M/M',
        8720,
        3,
        false,
        NOW() - INTERVAL '3 days'
    )
ON CONFLICT DO NOTHING;

COMMENT ON TABLE works IS 'Core fanfiction works with full metadata and statistics';
COMMENT ON TABLE tags IS 'Hierarchical tag system with canonical forms and synonyms';
COMMENT ON TABLE work_tags IS 'Many-to-many relationship between works and tags';
COMMENT ON FUNCTION update_work_stats() IS 'Automatically maintains work statistics when related data changes';
COMMENT ON VIEW published_works_with_stats IS 'Efficient query for published works with author and tag information';