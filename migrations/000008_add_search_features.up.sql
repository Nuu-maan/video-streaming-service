-- Enable pg_trgm extension for trigram-based similarity search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add tsvector column for full-text search
ALTER TABLE videos 
ADD COLUMN search_vector tsvector;

-- Create GIN index for fast full-text search
CREATE INDEX idx_videos_search_vector ON videos USING gin(search_vector);

-- Function to update search vector automatically
CREATE OR REPLACE FUNCTION videos_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update search vector on INSERT/UPDATE
CREATE TRIGGER videos_search_vector_trigger 
BEFORE INSERT OR UPDATE ON videos
FOR EACH ROW EXECUTE FUNCTION videos_search_vector_update();

-- Add categories, tags, and language columns
ALTER TABLE videos
ADD COLUMN category VARCHAR(50),
ADD COLUMN tags TEXT[],
ADD COLUMN language VARCHAR(10) DEFAULT 'en';

-- Create indexes for filtering
CREATE INDEX idx_videos_category ON videos(category);
CREATE INDEX idx_videos_tags ON videos USING gin(tags);
CREATE INDEX idx_videos_language ON videos(language);

-- Add denormalized counts for performance (view, like, comment counts)
ALTER TABLE videos
ADD COLUMN view_count BIGINT DEFAULT 0,
ADD COLUMN like_count BIGINT DEFAULT 0,
ADD COLUMN comment_count BIGINT DEFAULT 0,
ADD COLUMN share_count BIGINT DEFAULT 0;

-- Create indexes for sorting
CREATE INDEX idx_videos_view_count ON videos(view_count DESC);
CREATE INDEX idx_videos_like_count ON videos(like_count DESC);
CREATE INDEX idx_videos_created_at_desc ON videos(created_at DESC);

-- Add trigram index for autocomplete on titles
CREATE INDEX idx_videos_title_trgm ON videos USING gin(title gin_trgm_ops);

-- Update existing videos with empty search vector
UPDATE videos SET search_vector = 
    setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B')
WHERE search_vector IS NULL;
