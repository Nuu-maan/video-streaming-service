DROP INDEX IF EXISTS idx_videos_user_visibility;
DROP INDEX IF EXISTS idx_videos_visibility;
DROP INDEX IF EXISTS idx_videos_user_id;

ALTER TABLE videos
DROP COLUMN IF EXISTS visibility,
DROP COLUMN IF EXISTS user_id;

DROP TYPE IF EXISTS video_visibility;
