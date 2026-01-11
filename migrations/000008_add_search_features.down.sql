-- Rollback search features migration

-- Drop trigram index
DROP INDEX IF EXISTS idx_videos_title_trgm;

-- Drop sorting indexes
DROP INDEX IF EXISTS idx_videos_created_at_desc;
DROP INDEX IF EXISTS idx_videos_like_count;
DROP INDEX IF EXISTS idx_videos_view_count;

-- Remove denormalized count columns
ALTER TABLE videos
DROP COLUMN IF EXISTS share_count,
DROP COLUMN IF EXISTS comment_count,
DROP COLUMN IF EXISTS like_count,
DROP COLUMN IF EXISTS view_count;

-- Drop filter indexes
DROP INDEX IF EXISTS idx_videos_language;
DROP INDEX IF EXISTS idx_videos_tags;
DROP INDEX IF EXISTS idx_videos_category;

-- Remove category, tags, language columns
ALTER TABLE videos
DROP COLUMN IF EXISTS language,
DROP COLUMN IF EXISTS tags,
DROP COLUMN IF EXISTS category;

-- Drop search vector trigger and function
DROP TRIGGER IF EXISTS videos_search_vector_trigger ON videos;
DROP FUNCTION IF EXISTS videos_search_vector_update();

-- Drop search vector index
DROP INDEX IF EXISTS idx_videos_search_vector;

-- Remove search vector column
ALTER TABLE videos 
DROP COLUMN IF EXISTS search_vector;

-- Drop pg_trgm extension (optional, comment out if other tables use it)
-- DROP EXTENSION IF EXISTS pg_trgm;
