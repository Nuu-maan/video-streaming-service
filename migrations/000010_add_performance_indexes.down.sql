-- Rollback performance optimization indexes
-- Migration 000010_add_performance_indexes.down.sql

DROP INDEX IF EXISTS idx_videos_user_status;
DROP INDEX IF EXISTS idx_videos_visibility_status;
DROP INDEX IF EXISTS idx_videos_status_created;
DROP INDEX IF EXISTS idx_videos_view_count_desc;
DROP INDEX IF EXISTS idx_videos_like_count_desc;
DROP INDEX IF EXISTS idx_videos_created_status_visibility;
DROP INDEX IF EXISTS idx_comments_video_created;
DROP INDEX IF EXISTS idx_comments_video_likes;
DROP INDEX IF EXISTS idx_comments_parent_created;
DROP INDEX IF EXISTS idx_watch_history_user_watched;
DROP INDEX IF EXISTS idx_watch_history_video_count;
DROP INDEX IF EXISTS idx_notifications_user_read_created;
DROP INDEX IF EXISTS idx_notifications_user_unread;
DROP INDEX IF EXISTS idx_subscriptions_creator_created;
DROP INDEX IF EXISTS idx_likes_video_is_like;
DROP INDEX IF EXISTS idx_playlists_user_visibility;
DROP INDEX IF EXISTS idx_playlist_videos_position;
DROP INDEX IF EXISTS idx_video_views_video_created;
DROP INDEX IF EXISTS idx_video_views_user_video;
DROP INDEX IF EXISTS idx_content_reports_status_created;
DROP INDEX IF EXISTS idx_audit_logs_user_created;
DROP INDEX IF EXISTS idx_audit_logs_target_type_created;
DROP INDEX IF EXISTS idx_videos_processing;
DROP INDEX IF EXISTS idx_videos_pending;
DROP INDEX IF EXISTS idx_videos_list_covering;
