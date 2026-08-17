package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// stubAnalyticsVideoRepo serves one video by ID and reports a miss for anything
// else, which is what the real repository does.
type stubAnalyticsVideoRepo struct {
	video *domain.Video
}

func (r *stubAnalyticsVideoRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Video, error) {
	if r.video == nil || r.video.ID != id {
		return nil, domain.ErrVideoNotFound
	}
	return r.video, nil
}

// TestAuthorizeVideo pins who may read one video's analytics.
//
// The check runs before the cache is consulted, so a hole here is not merely a
// missing permission test — it would serve a cached payload for somebody else's
// video to the first caller who asked for it.
func TestAuthorizeVideo(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	videoID := uuid.New()

	owned := &domain.Video{ID: videoID, UserID: &owner}
	// Videos uploaded before authentication existed have no owner. Nobody may
	// claim them by being the caller.
	unowned := &domain.Video{ID: videoID}

	tests := []struct {
		name       string
		video      *domain.Video
		caller     uuid.UUID
		canViewAny bool
		wantErr    error
	}{
		{
			name:   "owner reads their own video",
			video:  owned,
			caller: owner,
		},
		{
			name:    "stranger is told the video does not exist",
			video:   owned,
			caller:  stranger,
			wantErr: domain.ErrVideoNotFound,
		},
		{
			name:       "view_analytics reads anyone's video",
			video:      owned,
			caller:     stranger,
			canViewAny: true,
		},
		{
			name:    "an unowned video is nobody's to read",
			video:   unowned,
			caller:  owner,
			wantErr: domain.ErrVideoNotFound,
		},
		{
			name:       "view_analytics still reads an unowned video",
			video:      unowned,
			caller:     stranger,
			canViewAny: true,
		},
		{
			name:    "a missing video is a miss, not a permission answer",
			video:   nil,
			caller:  owner,
			wantErr: domain.ErrVideoNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &AnalyticsService{videos: &stubAnalyticsVideoRepo{video: tt.video}}

			err := svc.authorizeVideo(context.Background(), videoID, tt.caller, tt.canViewAny)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("authorizeVideo = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestAuthorizeVideoDeniesBeforeReadingCache pins the ordering. GetVideoAnalytics
// holds a nil Redis client here: if the denial path ever moved after the cache
// lookup, this would panic instead of returning.
func TestAuthorizeVideoDeniesBeforeReadingCache(t *testing.T) {
	owner := uuid.New()
	videoID := uuid.New()

	svc := &AnalyticsService{
		videos: &stubAnalyticsVideoRepo{video: &domain.Video{ID: videoID, UserID: &owner}},
	}

	_, err := svc.GetVideoAnalytics(context.Background(), videoID, uuid.New(), false)
	if !errors.Is(err, domain.ErrVideoNotFound) {
		t.Fatalf("GetVideoAnalytics = %v, want %v", err, domain.ErrVideoNotFound)
	}

	_, err = svc.GetViewsTimeSeries(context.Background(), videoID, "hour", uuid.New(), false)
	if !errors.Is(err, domain.ErrVideoNotFound) {
		t.Fatalf("GetViewsTimeSeries = %v, want %v", err, domain.ErrVideoNotFound)
	}
}
