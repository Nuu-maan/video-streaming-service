-- Rollback social features migration

-- Drop triggers
DROP TRIGGER IF EXISTS update_playlist_updated_at ON playlists;
DROP TRIGGER IF EXISTS update_comment_updated_at ON comments;
DROP TRIGGER IF EXISTS update_playlist_video_count ON playlist_videos;
DROP TRIGGER IF EXISTS update_user_subscribers ON subscriptions;
DROP TRIGGER IF EXISTS update_comment_count ON comments;
DROP TRIGGER IF EXISTS update_like_count ON likes;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_playlist_timestamp();
DROP FUNCTION IF EXISTS update_comment_timestamp();
DROP FUNCTION IF EXISTS update_playlist_count();
DROP FUNCTION IF EXISTS update_subscriber_count();
DROP FUNCTION IF EXISTS update_video_counts();

-- Remove subscriber_count from users
DROP INDEX IF EXISTS idx_users_subscriber_count;
ALTER TABLE users DROP COLUMN IF EXISTS subscriber_count;

-- Drop notifications table and type
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_user;
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS notification_type;

-- Drop watch_later table
DROP INDEX IF EXISTS idx_watch_later_video;
DROP INDEX IF EXISTS idx_watch_later_user;
DROP TABLE IF EXISTS watch_later;

-- Drop watch_history table
DROP INDEX IF EXISTS idx_watch_history_video;
DROP INDEX IF EXISTS idx_watch_history_user;
DROP TABLE IF EXISTS watch_history;

-- Drop playlist_videos table
DROP INDEX IF EXISTS idx_playlist_videos_video;
DROP INDEX IF EXISTS idx_playlist_videos_playlist;
DROP TABLE IF EXISTS playlist_videos;

-- Drop playlists table
DROP INDEX IF EXISTS idx_playlists_visibility;
DROP INDEX IF EXISTS idx_playlists_user;
DROP TABLE IF EXISTS playlists;

-- Drop comments table
DROP INDEX IF EXISTS idx_comments_deleted;
DROP INDEX IF EXISTS idx_comments_parent;
DROP INDEX IF EXISTS idx_comments_user;
DROP INDEX IF EXISTS idx_comments_video;
DROP TABLE IF EXISTS comments;

-- Drop likes table
DROP INDEX IF EXISTS idx_likes_user_video;
DROP INDEX IF EXISTS idx_likes_user;
DROP INDEX IF EXISTS idx_likes_video;
DROP TABLE IF EXISTS likes;

-- Drop subscriptions table
DROP INDEX IF EXISTS idx_subscriptions_created_at;
DROP INDEX IF EXISTS idx_subscriptions_creator;
DROP INDEX IF EXISTS idx_subscriptions_subscriber;
DROP TABLE IF EXISTS subscriptions;
