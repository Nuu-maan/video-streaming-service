package domain

import (
	"time"

	"github.com/google/uuid"
)

type DashboardStats struct {
	TotalUsers        int64
	NewUsersToday     int64
	NewUsersThisWeek  int64
	ActiveUsers24h    int64
	TotalVideos       int64
	VideosToday       int64
	VideosThisWeek    int64
	ProcessingVideos  int64
	FailedVideos      int64
	TotalViews        int64
	ViewsToday        int64
	ViewsThisWeek     int64
	TotalStorageBytes int64
	StorageUsedGB     float64
	QueuedJobs        int64
	ActiveWorkers     int64
	PremiumUsers      int64
	MonthlyRevenue    float64
	LastUpdated       time.Time
}

type VideoAnalytics struct {
	VideoID          uuid.UUID
	Title            string
	UserID           uuid.UUID
	Username         string
	TotalViews       int64
	UniqueViewers    int64
	Likes            int64
	Dislikes         int64
	Comments         int64
	Shares           int64
	TotalWatchTime   int64
	AvgWatchTime     float64
	AvgWatchPercent  float64
	ViewsByQuality   map[string]int64
	AvgBufferTime    float64
	PlaybackErrors   int64
	SourceDirect     int64
	SourceSearch     int64
	SourceEmbed      int64
	SourceSocial     int64
	TopCountries     []CountryStats
	DeviceMobile     int64
	DeviceDesktop    int64
	DeviceTablet     int64
	CreatedAt        time.Time
	LastViewed       time.Time
}

type CountryStats struct {
	Country string
	Views   int64
}

type UserAnalytics struct {
	UserID           uuid.UUID
	Username         string
	VideosUploaded   int64
	TotalVideoViews  int64
	TotalWatchTime   int64
	TotalLikes       int64
	TotalComments    int64
	Subscribers      int64
	LastActive       time.Time
	DaysActive       int64
	AvgSessionTime   float64
	EstimatedRevenue float64
	CreatedAt        time.Time
}

type TimeSeriesData struct {
	Label      string
	Datapoints []DataPoint
}

type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

type RealtimeMetrics struct {
	ActiveViewers     int64
	UploadsLastHour   int64
	ViewsLastHour     int64
	CurrentCPU        float64
	CurrentMemory     float64
	QueuedJobs        int64
	ProcessingJobs    int64
	Timestamp         time.Time
}
