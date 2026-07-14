package domain

import (
	"time"

	"github.com/google/uuid"
)

// The types in this file are serialized straight to the API, so every field
// carries an explicit snake_case tag. Without one, encoding/json falls back to
// the Go field name and the analytics endpoints answer in PascalCase —
// {"TotalUsers": 3} — while every other endpoint in the service answers in
// snake_case. The serialized shape is part of the contract; it should not be an
// accident of Go naming.

type DashboardStats struct {
	TotalUsers        int64     `json:"total_users"`
	NewUsersToday     int64     `json:"new_users_today"`
	NewUsersThisWeek  int64     `json:"new_users_this_week"`
	ActiveUsers24h    int64     `json:"active_users_24h"`
	TotalVideos       int64     `json:"total_videos"`
	VideosToday       int64     `json:"videos_today"`
	VideosThisWeek    int64     `json:"videos_this_week"`
	ProcessingVideos  int64     `json:"processing_videos"`
	FailedVideos      int64     `json:"failed_videos"`
	TotalViews        int64     `json:"total_views"`
	ViewsToday        int64     `json:"views_today"`
	ViewsThisWeek     int64     `json:"views_this_week"`
	TotalStorageBytes int64     `json:"total_storage_bytes"`
	StorageUsedGB     float64   `json:"storage_used_gb"`
	QueuedJobs        int64     `json:"queued_jobs"`
	ActiveWorkers     int64     `json:"active_workers"`
	PremiumUsers      int64     `json:"premium_users"`
	MonthlyRevenue    float64   `json:"monthly_revenue"`
	LastUpdated       time.Time `json:"last_updated"`
}

type VideoAnalytics struct {
	VideoID         uuid.UUID        `json:"video_id"`
	Title           string           `json:"title"`
	UserID          uuid.UUID        `json:"user_id"`
	Username        string           `json:"username"`
	TotalViews      int64            `json:"total_views"`
	UniqueViewers   int64            `json:"unique_viewers"`
	Likes           int64            `json:"likes"`
	Dislikes        int64            `json:"dislikes"`
	Comments        int64            `json:"comments"`
	Shares          int64            `json:"shares"`
	TotalWatchTime  int64            `json:"total_watch_time"`
	AvgWatchTime    float64          `json:"avg_watch_time"`
	AvgWatchPercent float64          `json:"avg_watch_percent"`
	ViewsByQuality  map[string]int64 `json:"views_by_quality"`
	AvgBufferTime   float64          `json:"avg_buffer_time"`
	PlaybackErrors  int64            `json:"playback_errors"`
	SourceDirect    int64            `json:"source_direct"`
	SourceSearch    int64            `json:"source_search"`
	SourceEmbed     int64            `json:"source_embed"`
	SourceSocial    int64            `json:"source_social"`
	TopCountries    []CountryStats   `json:"top_countries"`
	DeviceMobile    int64            `json:"device_mobile"`
	DeviceDesktop   int64            `json:"device_desktop"`
	DeviceTablet    int64            `json:"device_tablet"`
	CreatedAt       time.Time        `json:"created_at"`
	LastViewed      time.Time        `json:"last_viewed"`
}

type CountryStats struct {
	Country string `json:"country"`
	Views   int64  `json:"views"`
}

type UserAnalytics struct {
	UserID           uuid.UUID `json:"user_id"`
	Username         string    `json:"username"`
	VideosUploaded   int64     `json:"videos_uploaded"`
	TotalVideoViews  int64     `json:"total_video_views"`
	TotalWatchTime   int64     `json:"total_watch_time"`
	TotalLikes       int64     `json:"total_likes"`
	TotalComments    int64     `json:"total_comments"`
	Subscribers      int64     `json:"subscribers"`
	LastActive       time.Time `json:"last_active"`
	DaysActive       int64     `json:"days_active"`
	AvgSessionTime   float64   `json:"avg_session_time"`
	EstimatedRevenue float64   `json:"estimated_revenue"`
	CreatedAt        time.Time `json:"created_at"`
}

type TimeSeriesData struct {
	Label      string      `json:"label"`
	Datapoints []DataPoint `json:"datapoints"`
}

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type RealtimeMetrics struct {
	ActiveViewers   int64     `json:"active_viewers"`
	UploadsLastHour int64     `json:"uploads_last_hour"`
	ViewsLastHour   int64     `json:"views_last_hour"`
	CurrentCPU      float64   `json:"current_cpu"`
	CurrentMemory   float64   `json:"current_memory"`
	QueuedJobs      int64     `json:"queued_jobs"`
	ProcessingJobs  int64     `json:"processing_jobs"`
	Timestamp       time.Time `json:"timestamp"`
}
