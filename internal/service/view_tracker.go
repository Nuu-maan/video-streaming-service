package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// ViewTrackerRepository is the slice of the analytics store this tracker needs.
// Satisfied by *postgres.AnalyticsRepository.
//
// RecordView does not take a watch duration: a view is durable the moment it
// starts, and the duration is only known once playback ends. The repository
// fills watch_duration/watch_percent afterwards via UpdateViewDuration.
type ViewTrackerRepository interface {
	RecordView(ctx context.Context, videoID, userID *uuid.UUID, sessionID, ipAddress, userAgent, quality, deviceType, country, source string) error
	UpsertWatchHistory(ctx context.Context, entry *domain.WatchHistory) error
	ListWatchHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.WatchHistory, int, error)
	ClearWatchHistory(ctx context.Context, userID uuid.UUID) error
	DeleteWatchHistoryEntry(ctx context.Context, userID, videoID uuid.UUID) error
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

// viewer returns the identity used to deduplicate views and count active
// viewers: the user ID when signed in, otherwise the session ID. It reports
// false when the event carries neither, in which case the viewer cannot be
// counted.
func (e ViewEvent) viewer() (string, bool) {
	if e.UserID != nil && *e.UserID != uuid.Nil {
		return e.UserID.String(), true
	}
	if e.SessionID != "" {
		return e.SessionID, true
	}
	return "", false
}

// viewDedupeWindow is how long repeat plays of the same video by the same
// viewer keep counting as the one original view.
const viewDedupeWindow = 30 * time.Minute

type ViewTracker struct {
	repo  ViewTrackerRepository
	redis *redis.Client
	log   *logger.Logger
}

func NewViewTracker(repo ViewTrackerRepository, redisClient *redis.Client, log *logger.Logger) *ViewTracker {
	return &ViewTracker{
		repo:  repo,
		redis: redisClient,
		log:   log,
	}
}

// TrackView records one view unless the same viewer already counted one for
// this video inside the dedupe window. It reports whether the view counted.
//
// Dedupe is a single SET NX EX in Redis. When Redis is unreachable the view is
// counted anyway: a view is not a security decision, and silently dropping
// real views during an outage is the worse failure. Worst case, a viewer
// double-counts while Redis is down.
func (vt *ViewTracker) TrackView(ctx context.Context, event ViewEvent) (bool, error) {
	viewer, ok := event.viewer()
	if !ok {
		return false, domain.ErrInvalidInput
	}

	dedupeKey := fmt.Sprintf("video:view_dedupe:%s:%s", event.VideoID, viewer)
	fresh, err := vt.redis.SetNX(ctx, dedupeKey, "1", viewDedupeWindow).Result()
	switch {
	case err != nil:
		vt.log.Warn(ctx, "view dedupe unavailable, counting view anyway", map[string]interface{}{
			"video_id": event.VideoID,
			"error":    err.Error(),
		})
	case !fresh:
		return false, nil
	}

	if err := vt.recordView(ctx, event); err != nil {
		// The dedupe key was claimed but no view was stored; release it
		// (best-effort) so the player's retry is not swallowed as a duplicate.
		vt.redis.Del(context.WithoutCancel(ctx), dedupeKey)
		return false, err
	}
	return true, nil
}

// recordView persists the view and refreshes the Redis counters behind the
// realtime analytics endpoints.
func (vt *ViewTracker) recordView(ctx context.Context, event ViewEvent) error {
	videoID := event.VideoID

	if err := vt.repo.RecordView(ctx, &videoID, event.UserID,
		event.SessionID, event.IPAddress, event.UserAgent,
		event.Quality, event.DeviceType, event.Country, event.Source,
	); err != nil {
		return fmt.Errorf("recording view in database: %w", err)
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

	if _, err := pipe.Exec(ctx); err != nil {
		// The durable row is already committed; these counters only accelerate
		// the realtime endpoints and rebuild themselves as views arrive, so
		// their failure must not fail a view that has in fact been counted.
		vt.log.Warn(ctx, "failed to update view counters in Redis", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
	}

	return nil
}

// SaveProgress upserts the caller's watch-history row for a video, which is
// what lets a frontend resume playback where the user left off. position and
// duration are in seconds; duration is the video's total length and only
// bounds position, it is not stored.
func (vt *ViewTracker) SaveProgress(ctx context.Context, userID, videoID uuid.UUID, position, duration int, completed bool) error {
	if position < 0 || duration < 0 || position > math.MaxInt32 {
		return domain.ErrInvalidProgress
	}
	if duration > 0 && position > duration {
		return domain.ErrInvalidProgress
	}

	entry := &domain.WatchHistory{
		UserID:        userID,
		VideoID:       videoID,
		WatchDuration: int32(position),
		Completed:     completed,
		LastPosition:  int32(position),
	}
	if err := entry.Validate(); err != nil {
		return err
	}

	if err := vt.repo.UpsertWatchHistory(ctx, entry); err != nil {
		return fmt.Errorf("saving watch progress: %w", err)
	}
	return nil
}

// History returns the user's watch history, most recent first, with the total
// for pagination.
func (vt *ViewTracker) History(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.WatchHistory, int, error) {
	entries, total, err := vt.repo.ListWatchHistory(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("loading watch history: %w", err)
	}
	return entries, total, nil
}

func (vt *ViewTracker) ClearHistory(ctx context.Context, userID uuid.UUID) error {
	if err := vt.repo.ClearWatchHistory(ctx, userID); err != nil {
		return fmt.Errorf("clearing watch history: %w", err)
	}
	return nil
}

func (vt *ViewTracker) RemoveHistoryEntry(ctx context.Context, userID, videoID uuid.UUID) error {
	return vt.repo.DeleteWatchHistoryEntry(ctx, userID, videoID)
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
