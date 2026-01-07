DROP TRIGGER IF EXISTS update_videos_updated_at ON videos;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_videos_title_search;
DROP INDEX IF EXISTS idx_videos_created_at;
DROP INDEX IF EXISTS idx_videos_status;
DROP TABLE IF EXISTS videos;
DROP TYPE IF EXISTS video_status;
DROP EXTENSION IF EXISTS "uuid-ossp";
