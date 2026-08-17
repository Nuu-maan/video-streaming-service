package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type AnalyticsRepository interface {
	GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error)
	GetVideoAnalytics(ctx context.Context, videoID uuid.UUID) (*domain.VideoAnalytics, error)
	GetUserAnalytics(ctx context.Context, userID uuid.UUID) (*domain.UserAnalytics, error)
	GetTopVideos(ctx context.Context, limit int, timeframe string) ([]*domain.VideoAnalytics, error)
	GetViewsTimeSeries(ctx context.Context, videoID uuid.UUID, interval string) (*domain.TimeSeriesData, error)
	GetRealtimeMetrics(ctx context.Context) (*domain.RealtimeMetrics, error)
}

// AnalyticsVideoRepository is the slice of the video store this service needs
// to resolve who owns a video. Satisfied by *postgres.PostgresVideoRepository.
type AnalyticsVideoRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error)
}

type AnalyticsService struct {
	repo   AnalyticsRepository
	videos AnalyticsVideoRepository
	redis  *redis.Client
}

func NewAnalyticsService(repo AnalyticsRepository, videos AnalyticsVideoRepository, redisClient *redis.Client) *AnalyticsService {
	return &AnalyticsService{
		repo:   repo,
		videos: videos,
		redis:  redisClient,
	}
}

// authorizeVideo decides who may read one video's analytics: its owner, or a
// caller holding view_analytics (the admin surface). Everyone else is told the
// video does not exist rather than that they may not see it, which is the rule
// the rest of the API follows — a video you cannot see must not be
// distinguishable from one that is not there.
//
// It runs before the cache is consulted, not after. The cache is keyed by video
// alone, so checking afterwards would hand a cached payload to the first
// unauthorised caller who asked.
func (s *AnalyticsService) authorizeVideo(ctx context.Context, videoID, userID uuid.UUID, canViewAny bool) error {
	video, err := s.videos.GetByID(ctx, videoID)
	if err != nil {
		return err
	}
	if canViewAny || video.IsOwnedBy(userID) {
		return nil
	}
	return domain.ErrVideoNotFound
}

func (s *AnalyticsService) GetDashboardOverview(ctx context.Context) (*domain.DashboardStats, error) {
	cacheKey := "analytics:dashboard:stats"

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var stats domain.DashboardStats
		if err := json.Unmarshal([]byte(cached), &stats); err == nil {
			return &stats, nil
		}
	}

	stats, err := s.repo.GetDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	statsJSON, err := json.Marshal(stats)
	if err == nil {
		s.redis.Set(ctx, cacheKey, statsJSON, 5*time.Minute)
	}

	return stats, nil
}

func (s *AnalyticsService) GetVideoAnalytics(ctx context.Context, videoID, userID uuid.UUID, canViewAny bool) (*domain.VideoAnalytics, error) {
	if err := s.authorizeVideo(ctx, videoID, userID, canViewAny); err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("analytics:video:%s", videoID)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var analytics domain.VideoAnalytics
		if err := json.Unmarshal([]byte(cached), &analytics); err == nil {
			return &analytics, nil
		}
	}

	analytics, err := s.repo.GetVideoAnalytics(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video analytics: %w", err)
	}

	analyticsJSON, err := json.Marshal(analytics)
	if err == nil {
		s.redis.Set(ctx, cacheKey, analyticsJSON, 1*time.Hour)
	}

	return analytics, nil
}

func (s *AnalyticsService) GetUserAnalytics(ctx context.Context, userID uuid.UUID) (*domain.UserAnalytics, error) {
	cacheKey := fmt.Sprintf("analytics:user:%s", userID)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var analytics domain.UserAnalytics
		if err := json.Unmarshal([]byte(cached), &analytics); err == nil {
			return &analytics, nil
		}
	}

	analytics, err := s.repo.GetUserAnalytics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user analytics: %w", err)
	}

	analyticsJSON, err := json.Marshal(analytics)
	if err == nil {
		s.redis.Set(ctx, cacheKey, analyticsJSON, 30*time.Minute)
	}

	return analytics, nil
}

func (s *AnalyticsService) GetTopVideosThisWeek(ctx context.Context, limit int) ([]*domain.VideoAnalytics, error) {
	cacheKey := fmt.Sprintf("analytics:top_videos:week:%d", limit)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var videos []*domain.VideoAnalytics
		if err := json.Unmarshal([]byte(cached), &videos); err == nil {
			return videos, nil
		}
	}

	videos, err := s.repo.GetTopVideos(ctx, limit, "week")
	if err != nil {
		return nil, fmt.Errorf("failed to get top videos: %w", err)
	}

	videosJSON, err := json.Marshal(videos)
	if err == nil {
		s.redis.Set(ctx, cacheKey, videosJSON, 1*time.Hour)
	}

	return videos, nil
}

func (s *AnalyticsService) GetViewsTimeSeries(ctx context.Context, videoID uuid.UUID, interval string, userID uuid.UUID, canViewAny bool) (*domain.TimeSeriesData, error) {
	if err := s.authorizeVideo(ctx, videoID, userID, canViewAny); err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("analytics:timeseries:%s:%s", videoID, interval)

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var timeseries domain.TimeSeriesData
		if err := json.Unmarshal([]byte(cached), &timeseries); err == nil {
			return &timeseries, nil
		}
	}

	timeseries, err := s.repo.GetViewsTimeSeries(ctx, videoID, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	timeseriesJSON, err := json.Marshal(timeseries)
	if err == nil {
		s.redis.Set(ctx, cacheKey, timeseriesJSON, 5*time.Minute)
	}

	return timeseries, nil
}

func (s *AnalyticsService) GetRealtimeMetrics(ctx context.Context) (*domain.RealtimeMetrics, error) {
	return s.repo.GetRealtimeMetrics(ctx)
}

func (s *AnalyticsService) InvalidateVideoCache(ctx context.Context, videoID uuid.UUID) error {
	cacheKey := fmt.Sprintf("analytics:video:%s", videoID)
	return s.redis.Del(ctx, cacheKey).Err()
}

func (s *AnalyticsService) InvalidateDashboardCache(ctx context.Context) error {
	return s.redis.Del(ctx, "analytics:dashboard:stats").Err()
}

func (s *AnalyticsService) InvalidateUserCache(ctx context.Context, userID uuid.UUID) error {
	cacheKey := fmt.Sprintf("analytics:user:%s", userID)
	return s.redis.Del(ctx, cacheKey).Err()
}
