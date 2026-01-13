-- Performance optimization indexes for video streaming platform
-- Migration 000010_add_performance_indexes.up.sql

-- Videos table performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_user_status 
ON videos(user_id, status) WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_visibility_status 
ON videos(visibility, status) WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_status_created 
ON videos(status, created_at DESC) WHERE visibility = 'public' AND deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_view_count_desc 
ON videos(view_count DESC) WHERE status = 'ready' AND visibility = 'public' AND deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_like_count_desc 
ON videos(like_count DESC) WHERE status = 'ready' AND visibility = 'public' AND deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_created_status_visibility 
ON videos(created_at DESC, status, visibility) WHERE deleted_at IS NULL;

-- Comments performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comments_video_created 
ON comments(video_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comments_video_likes 
ON comments(video_id, like_count DESC) WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comments_parent_created 
ON comments(parent_id, created_at DESC) WHERE deleted_at IS NULL AND parent_id IS NOT NULL;

-- Watch history performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_watch_history_user_watched 
ON watch_history(user_id, watched_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_watch_history_video_count 
ON watch_history(video_id);

-- Notifications performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_read_created 
ON notifications(user_id, read, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_unread 
ON notifications(user_id, created_at DESC) WHERE read = false;

-- Subscriptions performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_creator_created 
ON subscriptions(creator_id, created_at DESC);

-- Likes performance indexes  
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_likes_video_is_like 
ON likes(video_id, is_like) WHERE is_like = true;

-- Playlists performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_playlists_user_visibility 
ON playlists(user_id, visibility);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_playlist_videos_position 
ON playlist_videos(playlist_id, position);

-- Video views analytics index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_video_views_video_created 
ON video_views(video_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_video_views_user_video 
ON video_views(user_id, video_id) WHERE user_id IS NOT NULL;

-- Content reports performance index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_content_reports_status_created 
ON content_reports(status, created_at DESC);

-- Audit logs performance index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_admin_created 
ON audit_logs(admin_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_target_type_created 
ON audit_logs(target_type, created_at DESC);

-- Partial indexes for common queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_processing 
ON videos(id, created_at) WHERE status = 'processing';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_pending 
ON videos(id, created_at) WHERE status = 'pending';

-- Covering indexes for common read patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_videos_list_covering 
ON videos(created_at DESC, id, title, thumbnail_path, view_count, duration, user_id) 
WHERE status = 'ready' AND visibility = 'public' AND deleted_at IS NULL;

-- Statistics update for query planner
ANALYZE videos;
ANALYZE comments;
ANALYZE watch_history;
ANALYZE notifications;
ANALYZE subscriptions;
ANALYZE likes;
ANALYZE playlists;
ANALYZE playlist_videos;
ANALYZE video_views;
ANALYZE content_reports;
ANALYZE audit_logs;
