package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ViewTrackerRepository interface {
	RecordView(ctx context.Context, videoID, userID uuid.UUID, watchDuration int64) error
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

func (vt *ViewTracker) RecordView(ctx context.Context, videoID, userID uuid.UUID, watchDuration int64) error {
	if err := vt.repo.RecordView(ctx, videoID, userID, watchDuration); err != nil {
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
	pipe.ZAdd(ctx, activeViewersKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: userID.String(),
	})

	fiveMinutesAgo := now.Add(-5 * time.Minute).Unix()
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
