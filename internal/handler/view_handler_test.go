package handler

// Handler-level tests for ViewHandler's validation paths. The view tracker's
// Redis client is deliberately nil in every test here: each case must be
// rejected (or satisfied) before the dedupe/counter layer is reached, and a
// nil client turns any accidental reach into a loud panic instead of a
// silently green test.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
)

// stubViewRepo fakes service.ViewTrackerRepository, recording upserts.
type stubViewRepo struct {
	mu      sync.Mutex
	history []*domain.WatchHistory
}

func (r *stubViewRepo) RecordView(_ context.Context, _, _ *uuid.UUID, _, _, _, _, _, _, _ string) error {
	return nil
}

func (r *stubViewRepo) UpsertWatchHistory(_ context.Context, entry *domain.WatchHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = append(r.history, entry)
	return nil
}

func (r *stubViewRepo) ListWatchHistory(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.WatchHistory, int, error) {
	return nil, 0, nil
}

func (r *stubViewRepo) ClearWatchHistory(_ context.Context, _ uuid.UUID) error { return nil }

func (r *stubViewRepo) DeleteWatchHistoryEntry(_ context.Context, _, _ uuid.UUID) error {
	return domain.ErrWatchHistoryNotFound
}

func (r *stubViewRepo) entries() []*domain.WatchHistory {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*domain.WatchHistory(nil), r.history...)
}

func viewRouter(repo *stubViewRepo, principal *appctx.Principal) *gin.Engine {
	log := testLog()
	h := NewViewHandler(service.NewViewTracker(repo, nil, log), log)

	router := gin.New()
	if principal != nil {
		router.Use(withPrincipal(*principal))
	}
	router.POST("/videos/:id/progress", h.SaveProgress)
	router.POST("/videos/:id/view", h.RecordView)
	return router
}

func postJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestSaveProgressValidation pins the resume-position bounds: position and
// duration are non-negative seconds and position must lie within duration.
// A position past the end of the video is a 400, never stored.
func TestSaveProgressValidation(t *testing.T) {
	videoID := uuid.New()
	caller := principalFor(domain.RoleUser)

	tests := []struct {
		name       string
		principal  *appctx.Principal
		videoID    string
		body       string
		wantStatus int
		wantCode   string
		wantStored bool
	}{
		{
			name:       "valid progress is stored",
			principal:  caller,
			videoID:    videoID.String(),
			body:       `{"position": 42, "duration": 100, "completed": false}`,
			wantStatus: http.StatusOK,
			wantStored: true,
		},
		{
			name:       "position beyond duration is a 400",
			principal:  caller,
			videoID:    videoID.String(),
			body:       `{"position": 101, "duration": 100}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "negative position is a 400",
			principal:  caller,
			videoID:    videoID.String(),
			body:       `{"position": -5, "duration": 100}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "non-JSON body is a 400",
			principal:  caller,
			videoID:    videoID.String(),
			body:       `position=42`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "malformed video id is a 400",
			principal:  caller,
			videoID:    "not-a-uuid",
			body:       `{"position": 42, "duration": 100}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "anonymous caller is a 401",
			principal:  nil,
			videoID:    videoID.String(),
			body:       `{"position": 42, "duration": 100}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubViewRepo{}
			router := viewRouter(repo, tt.principal)

			rec := postJSON(t, router, "/videos/"+tt.videoID+"/progress", tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				env := decodeHandlerEnvelope(t, rec)
				if env.Error == nil || env.Error.Code != tt.wantCode {
					t.Errorf("error body = %s, want code %q", rec.Body.String(), tt.wantCode)
				}
			}

			stored := repo.entries()
			if tt.wantStored {
				if len(stored) != 1 {
					t.Fatalf("stored %d entries, want 1", len(stored))
				}
				if stored[0].LastPosition != 42 || stored[0].UserID != tt.principal.UserID {
					t.Errorf("stored entry = %+v, want position 42 for the caller", stored[0])
				}
			} else if len(stored) != 0 {
				t.Errorf("a rejected request stored %d entries; nothing may be written on a %d", len(stored), rec.Code)
			}
		})
	}
}

// TestRecordViewRequestValidation pins the anonymous-view contract: a view
// must carry a JSON body, and an anonymous one must carry a session_id, or
// the view cannot be deduplicated and is refused with 400 — before the
// tracker (and its Redis dedupe) is ever consulted.
func TestRecordViewRequestValidation(t *testing.T) {
	videoID := uuid.New()

	tests := []struct {
		name     string
		videoID  string
		body     string
		wantCode string
	}{
		{
			name:     "anonymous view without session_id",
			videoID:  videoID.String(),
			body:     `{}`,
			wantCode: "VALIDATION_ERROR",
		},
		{
			name:     "anonymous view with a blank session_id",
			videoID:  videoID.String(),
			body:     `{"session_id": "   "}`,
			wantCode: "VALIDATION_ERROR",
		},
		{
			name:     "non-JSON body",
			videoID:  videoID.String(),
			body:     `not json`,
			wantCode: "VALIDATION_ERROR",
		},
		{
			name:     "malformed video id",
			videoID:  "not-a-uuid",
			body:     `{"session_id": "abc"}`,
			wantCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := viewRouter(&stubViewRepo{}, nil)

			rec := postJSON(t, router, "/videos/"+tt.videoID+"/view", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
			}
			env := decodeHandlerEnvelope(t, rec)
			if env.Success {
				t.Error("success = true on a rejected view")
			}
			if env.Error == nil || env.Error.Code != tt.wantCode {
				t.Errorf("error body = %s, want code %q", rec.Body.String(), tt.wantCode)
			}
		})
	}
}
