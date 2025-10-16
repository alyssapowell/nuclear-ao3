-- Migration 015: Add Multi-Format Content Support
-- Adds support for HTML, Markdown, Plain Text, and Rich Text content formats

-- Add content format columns to chapters table
ALTER TABLE chapters 
ADD COLUMN content_format VARCHAR(20) DEFAULT 'html' NOT NULL,
ADD COLUMN raw_content TEXT, -- Store original unprocessed content
ADD COLUMN processed_content TEXT; -- Store processed/rendered content

-- Add constraint for content format values
ALTER TABLE chapters 
ADD CONSTRAINT content_format_values 
CHECK (content_format IN ('html', 'markdown', 'plain', 'richtext'));

-- Update existing chapters to use HTML format (AO3 default)
UPDATE chapters SET content_format = 'html' WHERE content_format IS NULL;

-- Add content format columns to works table for work-level summaries and notes
ALTER TABLE works 
ADD COLUMN summary_format VARCHAR(20) DEFAULT 'html' NOT NULL,
ADD COLUMN notes_format VARCHAR(20) DEFAULT 'html' NOT NULL;

-- Add constraints for work content format values
ALTER TABLE works 
ADD CONSTRAINT summary_format_values 
CHECK (summary_format IN ('html', 'markdown', 'plain', 'richtext')),
ADD CONSTRAINT notes_format_values 
CHECK (notes_format IN ('html', 'markdown', 'plain', 'richtext'));

-- Update existing works to use HTML format
UPDATE works SET summary_format = 'html' WHERE summary_format IS NULL;
UPDATE works SET notes_format = 'html' WHERE notes_format IS NULL;

-- Add content format support to comments
ALTER TABLE comments 
ADD COLUMN content_format VARCHAR(20) DEFAULT 'html' NOT NULL;

ALTER TABLE comments 
ADD CONSTRAINT comment_content_format_values 
CHECK (content_format IN ('html', 'markdown', 'plain', 'richtext'));

UPDATE comments SET content_format = 'html' WHERE content_format IS NULL;

-- Add indexes for content format filtering
CREATE INDEX idx_chapters_content_format ON chapters(content_format);
CREATE INDEX idx_works_summary_format ON works(summary_format);
CREATE INDEX idx_works_notes_format ON works(notes_format);
CREATE INDEX idx_comments_content_format ON comments(content_format);

-- Create content format statistics view
CREATE OR REPLACE VIEW content_format_stats AS
SELECT 
    'chapters' as content_type,
    content_format,
    COUNT(*) as usage_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (PARTITION BY 'chapters'), 2) as percentage
FROM chapters 
GROUP BY content_format
UNION ALL
SELECT 
    'works_summary' as content_type,
    summary_format as content_format,
    COUNT(*) as usage_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (PARTITION BY 'works_summary'), 2) as percentage
FROM works 
WHERE summary IS NOT NULL AND summary != ''
GROUP BY summary_format
UNION ALL
SELECT 
    'works_notes' as content_type,
    notes_format as content_format,
    COUNT(*) as usage_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (PARTITION BY 'works_notes'), 2) as percentage
FROM works 
WHERE notes IS NOT NULL AND notes != ''
GROUP BY notes_format
UNION ALL
SELECT 
    'comments' as content_type,
    content_format,
    COUNT(*) as usage_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (PARTITION BY 'comments'), 2) as percentage
FROM comments 
GROUP BY content_format
ORDER BY content_type, content_format;

-- Create function to convert between content formats
CREATE OR REPLACE FUNCTION convert_content_format(
    content_text TEXT,
    from_format VARCHAR(20),
    to_format VARCHAR(20)
) RETURNS TEXT
LANGUAGE plpgsql
AS $$
BEGIN
    -- Basic conversion logic (can be extended with proper conversion libraries)
    
    -- Convert from any format to plain text
    IF to_format = 'plain' THEN
        CASE from_format
            WHEN 'html' THEN
                -- Basic HTML tag removal (in production, use proper HTML parser)
                RETURN regexp_replace(
                    regexp_replace(content_text, '<[^>]*>', '', 'g'),
                    '\s+', ' ', 'g'
                );
            WHEN 'markdown' THEN
                -- Basic markdown removal
                RETURN regexp_replace(
                    regexp_replace(
                        regexp_replace(content_text, '\*\*([^*]+)\*\*', '\1', 'g'), -- Bold
                        '\*([^*]+)\*', '\1', 'g' -- Italic
                    ),
                    '#+ (.+)', '\1', 'g' -- Headers
                );
            WHEN 'richtext' THEN
                -- Treat richtext similar to HTML for now
                RETURN regexp_replace(
                    regexp_replace(content_text, '<[^>]*>', '', 'g'),
                    '\s+', ' ', 'g'
                );
            ELSE
                RETURN content_text;
        END CASE;
    END IF;
    
    -- Convert from plain to other formats
    IF from_format = 'plain' THEN
        CASE to_format
            WHEN 'html' THEN
                -- Convert line breaks to <br> and escape HTML
                RETURN replace(
                    replace(
                        replace(
                            replace(content_text, '&', '&amp;'),
                            '<', '&lt;'
                        ),
                        '>', '&gt;'
                    ),
                    E'\n', '<br>'
                );
            WHEN 'markdown' THEN
                -- Convert line breaks to markdown line breaks
                RETURN replace(content_text, E'\n', E'  \n');
            ELSE
                RETURN content_text;
        END CASE;
    END IF;
    
    -- For now, return original content if no conversion available
    RETURN content_text;
END;
$$;

-- Create function to update processed content for chapters
CREATE OR REPLACE FUNCTION update_processed_content()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Store the raw content
    NEW.raw_content = NEW.content;
    
    -- Process content based on format
    CASE NEW.content_format
        WHEN 'html' THEN
            -- HTML is already processed, store as-is
            NEW.processed_content = NEW.content;
        WHEN 'markdown' THEN
            -- In production, this would use a proper Markdown processor
            -- For now, do basic conversion
            NEW.processed_content = replace(
                replace(
                    replace(NEW.content, '**', '<strong>', 'g'),
                    '**', '</strong>', 'g'
                ),
                E'\n\n', '</p><p>'
            );
            NEW.processed_content = '<p>' || NEW.processed_content || '</p>';
        WHEN 'plain' THEN
            -- Convert plain text to HTML
            NEW.processed_content = convert_content_format(NEW.content, 'plain', 'html');
        WHEN 'richtext' THEN
            -- Rich text is already processed HTML
            NEW.processed_content = NEW.content;
        ELSE
            NEW.processed_content = NEW.content;
    END CASE;
    
    RETURN NEW;
END;
$$;

-- Create trigger to automatically process content on insert/update
CREATE TRIGGER trigger_update_processed_content
    BEFORE INSERT OR UPDATE OF content, content_format ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION update_processed_content();

-- Add helpful comments
COMMENT ON COLUMN chapters.content_format IS 'Format of the chapter content: html, markdown, plain, or richtext';
COMMENT ON COLUMN chapters.raw_content IS 'Original unprocessed content as entered by user';
COMMENT ON COLUMN chapters.processed_content IS 'Content processed for display (e.g., markdown converted to HTML)';
COMMENT ON COLUMN works.summary_format IS 'Format of the work summary';
COMMENT ON COLUMN works.notes_format IS 'Format of the work notes';
COMMENT ON COLUMN comments.content_format IS 'Format of the comment content';

COMMENT ON FUNCTION convert_content_format IS 'Convert content between different formats (basic implementation)';
COMMENT ON FUNCTION update_processed_content IS 'Automatically process chapter content based on format';
COMMENT ON VIEW content_format_stats IS 'Statistics on content format usage across the platform';

-- Create default content format preferences for users
ALTER TABLE users 
ADD COLUMN preferred_content_format VARCHAR(20) DEFAULT 'html';

ALTER TABLE users 
ADD CONSTRAINT user_preferred_content_format_values 
CHECK (preferred_content_format IN ('html', 'markdown', 'plain', 'richtext'));

UPDATE users SET preferred_content_format = 'html' WHERE preferred_content_format IS NULL;

COMMENT ON COLUMN users.preferred_content_format IS 'Users preferred content format for new works';