package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ViewTrackerRepository is the slice of the analytics store this tracker needs.
// Satisfied by *postgres.AnalyticsRepository.
//
// RecordView does not take a watch duration: a view is durable the moment it
// starts, and the duration is only known once playback ends. The repository
// fills watch_duration/watch_percent afterwards via UpdateViewDuration.
type ViewTrackerRepository interface {
	RecordView(ctx context.Context, videoID, userID *uuid.UUID, sessionID, ipAddress, userAgent, quality, deviceType, country, source string) error
}

// ViewEvent is one playback start, as recorded in video_views. UserID is nil for
// an anonymous viewer, who is identified by SessionID instead.
type ViewEvent struct {
	VideoID    uuid.UUID
	UserID     *uuid.UUID
	SessionID  string
	IPAddress  string
	UserAgent  string
	Quality    string
	DeviceType string
	Country    string
	Source     string
}

// viewer returns the identity used to count this event towards active viewers:
// the user ID when signed in, otherwise the session ID. It reports false when
// the event carries neither, in which case the viewer cannot be counted.
func (e ViewEvent) viewer() (string, bool) {
	if e.UserID != nil && *e.UserID != uuid.Nil {
		return e.UserID.String(), true
	}
	if e.SessionID != "" {
		return e.SessionID, true
	}
	return "", false
}

type ViewTracker struct {
	repo  ViewTrackerRepository
	redis *redis.Client
}

func NewViewTracker(repo ViewTrackerRepository, redisClient *redis.Client) *ViewTracker {
	return &ViewTracker{
		repo:  repo,
		redis: redisClient,
	}
}

func (vt *ViewTracker) RecordView(ctx context.Context, event ViewEvent) error {
	videoID := event.VideoID

	if err := vt.repo.RecordView(ctx, &videoID, event.UserID,
		event.SessionID, event.IPAddress, event.UserAgent,
		event.Quality, event.DeviceType, event.Country, event.Source,
	); err != nil {
		return fmt.Errorf("failed to record view in database: %w", err)
	}

	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	pipe := vt.redis.Pipeline()

	pipe.Incr(ctx, fmt.Sprintf("video:views:%s:total", videoID))

	todayKey := fmt.Sprintf("video:views:%s:today", videoID)
	pipe.Incr(ctx, todayKey)
	pipe.ExpireAt(ctx, todayKey, midnight)

	hourKey := fmt.Sprintf("video:views:%s:hour", videoID)
	pipe.Incr(ctx, hourKey)
	pipe.Expire(ctx, hourKey, 1*time.Hour)

	activeViewersKey := fmt.Sprintf("active_viewers:%s", videoID)
	fiveMinutesAgo := now.Add(-5 * time.Minute).Unix()

	if viewer, ok := event.viewer(); ok {
		pipe.ZAdd(ctx, activeViewersKey, redis.Z{
			Score:  float64(now.Unix()),
			Member: viewer,
		})
	}

	pipe.ZRemRangeByScore(ctx, activeViewersKey, "0", fmt.Sprintf("%d", fiveMinutesAgo))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update Redis counters: %w", err)
	}

	return nil
}

func (vt *ViewTracker) GetActiveViewers(ctx context.Context, videoID uuid.UUID) (int64, error) {
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute).Unix()
	activeViewersKey := fmt.Sprintf("active_viewers:%s", videoID)

	count, err := vt.redis.ZCount(ctx, activeViewersKey, fmt.Sprintf("%d", fiveMinutesAgo), "+inf").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get active viewers: %w", err)
	}

	return count, nil
}

func (vt *ViewTracker) GetTodayViews(ctx context.Context, videoID uuid.UUID) (int64, error) {
	todayKey := fmt.Sprintf("video:views:%s:today", videoID)
	count, err := vt.redis.Get(ctx, todayKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get today's views: %w", err)
	}
	return count, nil
}

func (vt *ViewTracker) GetHourViews(ctx context.Context, videoID uuid.UUID) (int64, error) {
	hourKey := fmt.Sprintf("video:views:%s:hour", videoID)
	count, err := vt.redis.Get(ctx, hourKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get hour's views: %w", err)
	}
	return count, nil
}
