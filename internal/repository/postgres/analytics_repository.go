package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgForeignKeyViolation is the SQLSTATE for a foreign-key violation, matched
// by code (never by message text) like the unique-violation handling in
// user_repository.go.
const pgForeignKeyViolation = "23503"

// isVideoFKViolation reports whether err is the video_id foreign key failing,
// i.e. the referenced video does not exist. The constraint name is schema
// metadata, not localized driver prose, so matching on it is stable.
func isVideoFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgForeignKeyViolation &&
		strings.Contains(pgErr.ConstraintName, "video_id")
}

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	query := `
	WITH user_stats AS (
		SELECT 
			COUNT(*) as total_users,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '1 day' THEN 1 END) as new_today,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as new_week,
			COUNT(CASE WHEN last_login_at >= NOW() - INTERVAL '1 day' THEN 1 END) as active_24h
		FROM users
		WHERE deleted_at IS NULL
	),
	video_stats AS (
		SELECT 
			COUNT(*) as total_videos,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '1 day' THEN 1 END) as videos_today,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as videos_week,
			COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COALESCE(SUM(file_size), 0) as total_storage
		FROM videos
	),
	view_stats AS (
		SELECT 
			COUNT(*) as total_views,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '1 day' THEN 1 END) as views_today,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as views_week
		FROM video_views
	)
	SELECT * FROM user_stats, video_stats, view_stats;
	`

	stats := &domain.DashboardStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.NewUsersToday,
		&stats.NewUsersThisWeek,
		&stats.ActiveUsers24h,
		&stats.TotalVideos,
		&stats.VideosToday,
		&stats.VideosThisWeek,
		&stats.ProcessingVideos,
		&stats.FailedVideos,
		&stats.TotalStorageBytes,
		&stats.TotalViews,
		&stats.ViewsToday,
		&stats.ViewsThisWeek,
	)
	if err != nil {
		return nil, err
	}

	stats.StorageUsedGB = float64(stats.TotalStorageBytes) / (1024 * 1024 * 1024)
	stats.LastUpdated = time.Now()

	return stats, nil
}

func (r *AnalyticsRepository) GetVideoAnalytics(ctx context.Context, videoID uuid.UUID) (*domain.VideoAnalytics, error) {
	query := `
	SELECT 
		v.id,
		v.title,
		v.user_id,
		u.username,
		COALESCE(COUNT(DISTINCT vv.id), 0) as total_views,
		COALESCE(COUNT(DISTINCT vv.user_id), 0) as unique_viewers,
		COALESCE(SUM(vv.watch_duration), 0) as total_watch_time,
		COALESCE(AVG(vv.watch_duration), 0) as avg_watch_time,
		COALESCE(AVG(vv.watch_percent), 0) as avg_watch_percent,
		v.created_at,
		COALESCE(MAX(vv.created_at), v.created_at) as last_viewed
	FROM videos v
	LEFT JOIN users u ON v.user_id = u.id
	LEFT JOIN video_views vv ON v.id = vv.video_id
	WHERE v.id = $1
	GROUP BY v.id, v.title, v.user_id, u.username, v.created_at
	`

	analytics := &domain.VideoAnalytics{
		VideoID:        videoID,
		ViewsByQuality: make(map[string]int64),
		TopCountries:   []domain.CountryStats{},
	}

	err := r.db.QueryRow(ctx, query, videoID).Scan(
		&analytics.VideoID,
		&analytics.Title,
		&analytics.UserID,
		&analytics.Username,
		&analytics.TotalViews,
		&analytics.UniqueViewers,
		&analytics.TotalWatchTime,
		&analytics.AvgWatchTime,
		&analytics.AvgWatchPercent,
		&analytics.CreatedAt,
		&analytics.LastViewed,
	)
	if err != nil {
		// Analytics for a video that does not exist is a client mistake, so it has
		// to be distinguishable from a real query failure by the HTTP layer.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVideoNotFound
		}
		return nil, fmt.Errorf("getting analytics for video %s: %w", videoID, err)
	}

	qualityStats, err := r.GetPopularQualities(ctx, videoID)
	if err == nil {
		analytics.ViewsByQuality = qualityStats
	}

	deviceQuery := `
	SELECT 
		COUNT(CASE WHEN device_type = 'mobile' THEN 1 END) as mobile,
		COUNT(CASE WHEN device_type = 'desktop' THEN 1 END) as desktop,
		COUNT(CASE WHEN device_type = 'tablet' THEN 1 END) as tablet
	FROM video_views
	WHERE video_id = $1
	`

	_ = r.db.QueryRow(ctx, deviceQuery, videoID).Scan(
		&analytics.DeviceMobile,
		&analytics.DeviceDesktop,
		&analytics.DeviceTablet,
	)

	sourceQuery := `
	SELECT 
		COUNT(CASE WHEN source = 'direct' THEN 1 END) as direct,
		COUNT(CASE WHEN source = 'search' THEN 1 END) as search,
		COUNT(CASE WHEN source = 'embed' THEN 1 END) as embed,
		COUNT(CASE WHEN source = 'social' THEN 1 END) as social
	FROM video_views
	WHERE video_id = $1
	`

	_ = r.db.QueryRow(ctx, sourceQuery, videoID).Scan(
		&analytics.SourceDirect,
		&analytics.SourceSearch,
		&analytics.SourceEmbed,
		&analytics.SourceSocial,
	)

	countries, err := r.GetGeographyStats(ctx, videoID)
	if err == nil {
		analytics.TopCountries = countries
	}

	return analytics, nil
}

func (r *AnalyticsRepository) GetTopVideos(ctx context.Context, limit int, timeframe string) ([]*domain.VideoAnalytics, error) {
	timeFilter := ""
	switch timeframe {
	case "today":
		timeFilter = "AND vv.created_at >= NOW() - INTERVAL '1 day'"
	case "week":
		timeFilter = "AND vv.created_at >= NOW() - INTERVAL '7 days'"
	case "month":
		timeFilter = "AND vv.created_at >= NOW() - INTERVAL '30 days'"
	}

	query := `
	SELECT 
		v.id,
		v.title,
		v.user_id,
		u.username,
		COUNT(DISTINCT vv.id) as total_views,
		COUNT(DISTINCT vv.user_id) as unique_viewers,
		COALESCE(SUM(vv.watch_duration), 0) as total_watch_time,
		v.created_at
	FROM videos v
	LEFT JOIN users u ON v.user_id = u.id
	LEFT JOIN video_views vv ON v.id = vv.video_id ` + timeFilter + `
	WHERE v.status = 'ready'
	GROUP BY v.id, v.title, v.user_id, u.username, v.created_at
	ORDER BY total_views DESC
	LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*domain.VideoAnalytics
	for rows.Next() {
		video := &domain.VideoAnalytics{
			ViewsByQuality: make(map[string]int64),
		}
		err := rows.Scan(
			&video.VideoID,
			&video.Title,
			&video.UserID,
			&video.Username,
			&video.TotalViews,
			&video.UniqueViewers,
			&video.TotalWatchTime,
			&video.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func (r *AnalyticsRepository) GetUserAnalytics(ctx context.Context, userID uuid.UUID) (*domain.UserAnalytics, error) {
	query := `
	SELECT 
		u.id,
		u.username,
		COUNT(DISTINCT v.id) as videos_uploaded,
		COALESCE(SUM(view_counts.total_views), 0) as total_video_views,
		COALESCE(SUM(view_counts.total_watch_time), 0) as total_watch_time,
		u.created_at,
		u.last_login_at
	FROM users u
	LEFT JOIN videos v ON u.id = v.user_id
	LEFT JOIN (
		SELECT video_id, COUNT(*) as total_views, SUM(watch_duration) as total_watch_time
		FROM video_views
		GROUP BY video_id
	) view_counts ON v.id = view_counts.video_id
	WHERE u.id = $1
	GROUP BY u.id, u.username, u.created_at, u.last_login_at
	`

	analytics := &domain.UserAnalytics{UserID: userID}
	var lastLogin sql.NullTime

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&analytics.UserID,
		&analytics.Username,
		&analytics.VideosUploaded,
		&analytics.TotalVideoViews,
		&analytics.TotalWatchTime,
		&analytics.CreatedAt,
		&lastLogin,
	)
	if err != nil {
		return nil, err
	}

	if lastLogin.Valid {
		analytics.LastActive = lastLogin.Time
	}

	return analytics, nil
}

func (r *AnalyticsRepository) GetViewsTimeSeries(ctx context.Context, videoID uuid.UUID, interval string) (*domain.TimeSeriesData, error) {
	truncFunc := "hour"
	intervalDuration := "7 days"

	switch interval {
	case "day":
		truncFunc = "day"
		intervalDuration = "30 days"
	case "week":
		truncFunc = "week"
		intervalDuration = "90 days"
	case "month":
		truncFunc = "month"
		intervalDuration = "365 days"
	}

	query := `
	SELECT 
		DATE_TRUNC($1, created_at) as timestamp,
		COUNT(*) as views
	FROM video_views
	WHERE video_id = $2
		AND created_at >= NOW() - INTERVAL '` + intervalDuration + `'
	GROUP BY DATE_TRUNC($1, created_at)
	ORDER BY timestamp ASC
	`

	rows, err := r.db.Query(ctx, query, truncFunc, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := &domain.TimeSeriesData{
		Label:      "Views",
		Datapoints: []domain.DataPoint{},
	}

	for rows.Next() {
		var dp domain.DataPoint
		err := rows.Scan(&dp.Timestamp, &dp.Value)
		if err != nil {
			return nil, err
		}
		data.Datapoints = append(data.Datapoints, dp)
	}

	return data, nil
}

func (r *AnalyticsRepository) GetPopularQualities(ctx context.Context, videoID uuid.UUID) (map[string]int64, error) {
	query := `
	SELECT quality, COUNT(*) as count
	FROM video_views
	WHERE video_id = $1 AND quality IS NOT NULL
	GROUP BY quality
	ORDER BY count DESC
	`

	rows, err := r.db.Query(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	qualities := make(map[string]int64)
	for rows.Next() {
		var quality string
		var count int64
		err := rows.Scan(&quality, &count)
		if err != nil {
			return nil, err
		}
		qualities[quality] = count
	}

	return qualities, nil
}

func (r *AnalyticsRepository) GetGeographyStats(ctx context.Context, videoID uuid.UUID) ([]domain.CountryStats, error) {
	query := `
	SELECT country, COUNT(*) as views
	FROM video_views
	WHERE video_id = $1 AND country IS NOT NULL
	GROUP BY country
	ORDER BY views DESC
	LIMIT 10
	`

	rows, err := r.db.Query(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []domain.CountryStats
	for rows.Next() {
		var country domain.CountryStats
		err := rows.Scan(&country.Country, &country.Views)
		if err != nil {
			return nil, err
		}
		countries = append(countries, country)
	}

	return countries, nil
}

// GetRealtimeMetrics reports the last-hour activity snapshot the admin dashboard
// polls for.
//
// A viewer counts as active if they produced a video_views row in the last five
// minutes; anonymous viewers have a NULL user_id, so they are identified by
// session_id instead and each distinct session counts once.
//
// CurrentCPU and CurrentMemory are deliberately left at zero: they are process
// metrics, not database state, and nothing in the schema records them. The
// monitoring service is the source for those.
func (r *AnalyticsRepository) GetRealtimeMetrics(ctx context.Context) (*domain.RealtimeMetrics, error) {
	query := `
	SELECT
		(SELECT COUNT(DISTINCT COALESCE(user_id::text, session_id))
		   FROM video_views
		  WHERE created_at >= NOW() - INTERVAL '5 minutes'
		    AND (user_id IS NOT NULL OR session_id IS NOT NULL)) AS active_viewers,
		(SELECT COUNT(*) FROM videos
		  WHERE created_at >= NOW() - INTERVAL '1 hour') AS uploads_last_hour,
		(SELECT COUNT(*) FROM video_views
		  WHERE created_at >= NOW() - INTERVAL '1 hour') AS views_last_hour,
		(SELECT COUNT(*) FROM videos WHERE status = $1) AS queued_jobs,
		(SELECT COUNT(*) FROM videos WHERE status = $2) AS processing_jobs
	`

	metrics := &domain.RealtimeMetrics{}
	err := r.db.QueryRow(ctx, query, domain.VideoStatusUploading, domain.VideoStatusProcessing).Scan(
		&metrics.ActiveViewers,
		&metrics.UploadsLastHour,
		&metrics.ViewsLastHour,
		&metrics.QueuedJobs,
		&metrics.ProcessingJobs,
	)
	if err != nil {
		return nil, fmt.Errorf("getting realtime metrics: %w", err)
	}

	metrics.Timestamp = time.Now()
	return metrics, nil
}

func (r *AnalyticsRepository) RecordView(ctx context.Context, videoID, userID *uuid.UUID, sessionID, ipAddress, userAgent, quality, deviceType, country, source string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning view transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// NULLIF keeps absent optionals out of the analytics aggregates, which
	// filter on IS NOT NULL; inserting '' would surface as a phantom country
	// or quality bucket in the per-video breakdowns.
	insert := `
	INSERT INTO video_views (video_id, user_id, session_id, ip_address, user_agent, quality, device_type, country, source)
	VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''))
	`
	if _, err := tx.Exec(ctx, insert, videoID, userID, sessionID, ipAddress, userAgent, quality, deviceType, country, source); err != nil {
		if isVideoFKViolation(err) {
			return domain.ErrVideoNotFound
		}
		return fmt.Errorf("recording view for video %s: %w", videoID, err)
	}

	// No trigger maintains videos.view_count from video_views — the 000009
	// count triggers cover likes, comments, subscriptions, and playlists only —
	// so the denormalised counter is bumped here, inside the same transaction
	// as the raw view row, to keep the two from drifting.
	if _, err := tx.Exec(ctx, `UPDATE videos SET view_count = view_count + 1 WHERE id = $1`, videoID); err != nil {
		return fmt.Errorf("incrementing view count for video %s: %w", videoID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing view for video %s: %w", videoID, err)
	}
	return nil
}

// UpsertWatchHistory records playback progress, replacing the user's previous
// entry for the video (UNIQUE(user_id, video_id)). last_position always tracks
// the latest report so resume-playback works after seeking backwards, while
// watch_duration only grows and completed only latches on, so rewatching the
// intro cannot shrink how much of the video the user is credited with.
func (r *AnalyticsRepository) UpsertWatchHistory(ctx context.Context, entry *domain.WatchHistory) error {
	query := `
	INSERT INTO watch_history (user_id, video_id, watch_duration, completed, last_position, watched_at)
	VALUES ($1, $2, $3, $4, $5, NOW())
	ON CONFLICT (user_id, video_id) DO UPDATE SET
		watch_duration = GREATEST(watch_history.watch_duration, EXCLUDED.watch_duration),
		completed = watch_history.completed OR EXCLUDED.completed,
		last_position = EXCLUDED.last_position,
		watched_at = NOW()
	`

	if _, err := r.db.Exec(ctx, query, entry.UserID, entry.VideoID, entry.WatchDuration, entry.Completed, entry.LastPosition); err != nil {
		if isVideoFKViolation(err) {
			return domain.ErrVideoNotFound
		}
		return fmt.Errorf("upserting watch history for user %s video %s: %w", entry.UserID, entry.VideoID, err)
	}
	return nil
}

// ListWatchHistory returns userID's watch history, most recently watched
// first, along with the total row count for pagination.
func (r *AnalyticsRepository) ListWatchHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.WatchHistory, int, error) {
	query := `
	SELECT id, user_id, video_id, watched_at, watch_duration, completed, last_position
	FROM watch_history
	WHERE user_id = $1
	ORDER BY watched_at DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing watch history for user %s: %w", userID, err)
	}
	defer rows.Close()

	var entries []*domain.WatchHistory
	for rows.Next() {
		entry := &domain.WatchHistory{}
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.VideoID, &entry.WatchedAt,
			&entry.WatchDuration, &entry.Completed, &entry.LastPosition,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning watch history row: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating watch history: %w", err)
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM watch_history WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting watch history for user %s: %w", userID, err)
	}

	return entries, total, nil
}

func (r *AnalyticsRepository) ClearWatchHistory(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM watch_history WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clearing watch history for user %s: %w", userID, err)
	}
	return nil
}

func (r *AnalyticsRepository) DeleteWatchHistoryEntry(ctx context.Context, userID, videoID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM watch_history WHERE user_id = $1 AND video_id = $2`, userID, videoID)
	if err != nil {
		return fmt.Errorf("deleting watch history entry for user %s video %s: %w", userID, videoID, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrWatchHistoryNotFound
	}
	return nil
}

func (r *AnalyticsRepository) UpdateViewDuration(ctx context.Context, viewID uuid.UUID, duration int, percent float64) error {
	query := `
	UPDATE video_views
	SET watch_duration = $1, watch_percent = $2
	WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, duration, percent, viewID)
	return err
}
