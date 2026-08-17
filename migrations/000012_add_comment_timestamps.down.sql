DROP INDEX IF EXISTS idx_comments_video_timestamp;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS comments_video_timestamp_non_negative;
ALTER TABLE comments DROP COLUMN IF EXISTS video_timestamp;
