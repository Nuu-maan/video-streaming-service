package handler

// Handler-level tests for VideoHandler, driven through gin with the repository
// interface faked in memory. They pin the same contracts the app-level
// integration tests pin, but at the layer that owns them, so a router
// refactor cannot silently drop the handler's own checks. No Postgres and no
// Redis are involved; nothing here skips.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/internal/storage"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testLog() *logger.Logger {
	return logger.New("production", "error")
}

// withPrincipal attaches an authenticated caller to the request context, the
// way the auth middleware would after validating a token.
func withPrincipal(p appctx.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(appctx.WithPrincipal(c.Request.Context(), p))
	}
}

// stubVideoRepo is an in-memory repository.VideoRepository that records the
// filter it was last queried with, so tests can assert what the handler asked
// for — not just what came back.
type stubVideoRepo struct {
	mu         sync.Mutex
	videos     map[uuid.UUID]*domain.Video
	lastFilter repository.VideoFilter
}

func newStubVideoRepo() *stubVideoRepo {
	return &stubVideoRepo{videos: make(map[uuid.UUID]*domain.Video)}
}

func (r *stubVideoRepo) add(v *domain.Video) *domain.Video {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.videos[v.ID] = v
	return v
}

func (r *stubVideoRepo) has(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.videos[id]
	return ok
}

func (r *stubVideoRepo) filterSeen() repository.VideoFilter {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastFilter
}

func (r *stubVideoRepo) Create(_ context.Context, v *domain.Video) error {
	r.add(v)
	return nil
}

func (r *stubVideoRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.videos[id]
	if !ok {
		return nil, domain.ErrVideoNotFound
	}
	return v, nil
}

func (r *stubVideoRepo) matching(filter repository.VideoFilter) []*domain.Video {
	var out []*domain.Video
	for _, v := range r.videos {
		if filter.Status != nil && v.Status != *filter.Status {
			continue
		}
		if filter.OwnerID != nil && (v.UserID == nil || *v.UserID != *filter.OwnerID) {
			continue
		}
		if filter.Visibility != nil && v.Visibility != *filter.Visibility {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (r *stubVideoRepo) List(_ context.Context, filter repository.VideoFilter, page repository.Page) ([]*domain.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastFilter = filter
	all := r.matching(filter)
	if page.Offset >= len(all) {
		return nil, nil
	}
	end := page.Offset + page.Limit
	if end > len(all) {
		end = len(all)
	}
	return all[page.Offset:end], nil
}

func (r *stubVideoRepo) Count(_ context.Context, filter repository.VideoFilter) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.matching(filter)), nil
}

func (r *stubVideoRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.videos[id]; !ok {
		return domain.ErrVideoNotFound
	}
	delete(r.videos, id)
	return nil
}

func (r *stubVideoRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.VideoStatus) error {
	return nil
}
func (r *stubVideoRepo) UpdateProgress(_ context.Context, _ uuid.UUID, _ int) error      { return nil }
func (r *stubVideoRepo) UpdateDuration(_ context.Context, _ uuid.UUID, _ int) error      { return nil }
func (r *stubVideoRepo) UpdateResolution(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *stubVideoRepo) UpdateHLSInfo(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
	return nil
}
func (r *stubVideoRepo) MarkAsReady(_ context.Context, _ uuid.UUID, _ []string, _ string) error {
	return nil
}
func (r *stubVideoRepo) MarkAsFailed(_ context.Context, _ uuid.UUID) error { return nil }

// nullStore is a storage.Store for code paths that must tolerate storage but
// never depend on its contents (best-effort file cleanup after a delete).
type nullStore struct{}

func (nullStore) Save(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error { return nil }
func (nullStore) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	return nil, fmt.Errorf("no object %q", key)
}
func (nullStore) Stat(_ context.Context, key string) (storage.FileInfo, error) {
	return storage.FileInfo{}, fmt.Errorf("no object %q", key)
}
func (nullStore) Delete(_ context.Context, _ string) error       { return nil }
func (nullStore) DeletePrefix(_ context.Context, _ string) error { return nil }
func (nullStore) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// newTestVideoHandler builds a VideoHandler over the stub repository. The
// queue client is nil — no test here reaches the enqueue path.
func newTestVideoHandler(repo *stubVideoRepo) *VideoHandler {
	log := testLog()
	cfg := &config.Config{}
	uploadSvc := service.NewUploadService(repo, service.NewFFmpegService(log), &cfg.Storage, nullStore{}, log)
	return NewVideoHandler(uploadSvc, repo, nil, log, cfg)
}

// videoRouter mounts the handler's read/delete routes, optionally behind an
// authenticated principal.
func videoRouter(h *VideoHandler, principal *appctx.Principal) *gin.Engine {
	router := gin.New()
	if principal != nil {
		router.Use(withPrincipal(*principal))
	}
	router.GET("/videos", h.ListVideos)
	router.GET("/videos/:id", h.GetVideo)
	router.DELETE("/videos/:id", h.DeleteVideo)
	return router
}

// handlerEnvelope mirrors the API's response envelope for assertions.
type handlerEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Pagination *response.PaginationMeta `json:"pagination"`
}

func decodeHandlerEnvelope(t *testing.T, rec *httptest.ResponseRecorder) handlerEnvelope {
	t.Helper()
	var env handlerEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response is not the JSON envelope: %v (body: %s)", err, rec.Body.String())
	}
	return env
}

func seedVideo(repo *stubVideoRepo, owner uuid.UUID, visibility domain.VideoVisibility) *domain.Video {
	id := uuid.New()
	ownerID := owner
	return repo.add(&domain.Video{
		ID:         id,
		UserID:     &ownerID,
		Title:      "video-" + id.String()[:8],
		Filename:   "source.mp4",
		FileSize:   1024,
		MimeType:   "video/mp4",
		Status:     domain.VideoStatusReady,
		Visibility: visibility,
	})
}

func principalFor(role domain.Role) *appctx.Principal {
	return &appctx.Principal{UserID: uuid.New(), Username: "caller", Role: role}
}

// TestGetVideoVisibility pins canViewVideo's contract at the handler: a
// private video resolves only for its owner or a watch_private holder, and
// denial is always 404 NOT_FOUND — never 403, which would confirm the video
// exists.
//
// Regression: PermissionWatchPrivate used to be checked nowhere, so a private
// video was readable by anyone who had its ID.
func TestGetVideoVisibility(t *testing.T) {
	owner := appctx.Principal{UserID: uuid.New(), Username: "owner", Role: domain.RoleUser}

	tests := []struct {
		name       string
		visibility domain.VideoVisibility
		caller     *appctx.Principal
		wantStatus int
	}{
		{
			name:       "private video hidden from anonymous",
			visibility: domain.VisibilityPrivate,
			caller:     nil,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "private video hidden from another user",
			visibility: domain.VisibilityPrivate,
			caller:     principalFor(domain.RoleUser),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "private video visible to its owner",
			visibility: domain.VisibilityPrivate,
			caller:     &owner,
			wantStatus: http.StatusOK,
		},
		{
			name:       "private video visible to a watch_private holder",
			visibility: domain.VisibilityPrivate,
			caller:     principalFor(domain.RolePremium),
			wantStatus: http.StatusOK,
		},
		{
			name:       "unlisted video reachable by link anonymously",
			visibility: domain.VisibilityUnlisted,
			caller:     nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "public video visible to anonymous",
			visibility: domain.VisibilityPublic,
			caller:     nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newStubVideoRepo()
			video := seedVideo(repo, owner.UserID, tt.visibility)
			router := videoRouter(newTestVideoHandler(repo), tt.caller)

			req := httptest.NewRequest(http.MethodGet, "/videos/"+video.ID.String(), nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Fatalf("denial was 403; it must be 404 so it does not reveal the video exists (body: %s)", rec.Body.String())
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusNotFound {
				env := decodeHandlerEnvelope(t, rec)
				if env.Error == nil || env.Error.Code != "NOT_FOUND" {
					t.Errorf("denial body = %s, want error code NOT_FOUND", rec.Body.String())
				}
			}
		})
	}
}

// TestGetVideoBadID pins the request-validation half of the contract: a
// malformed UUID is a 400 VALIDATION_ERROR in the standard error envelope, and
// an unknown-but-well-formed one is a 404.
func TestGetVideoBadID(t *testing.T) {
	router := videoRouter(newTestVideoHandler(newStubVideoRepo()), nil)

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantCode   string
	}{
		{name: "malformed UUID", id: "not-a-uuid", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "unknown UUID", id: uuid.NewString(), wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/videos/"+tt.id, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			env := decodeHandlerEnvelope(t, rec)
			if env.Success {
				t.Error("success = true on an error response")
			}
			if env.Error == nil || env.Error.Code != tt.wantCode {
				t.Errorf("error body = %s, want code %q", rec.Body.String(), tt.wantCode)
			}
		})
	}
}

// TestListVideosPagination pins that the pagination envelope is derived from
// the repository's Count over the same filter as the page.
//
// Regression: the handler used to report len(currentPage) as the total, so
// total_pages was always 1 and has_next was always false.
func TestListVideosPagination(t *testing.T) {
	repo := newStubVideoRepo()
	creator := uuid.New()
	for i := 0; i < 7; i++ {
		seedVideo(repo, creator, domain.VisibilityPublic)
	}
	for i := 0; i < 2; i++ {
		seedVideo(repo, creator, domain.VisibilityPrivate)
	}

	router := videoRouter(newTestVideoHandler(repo), nil)

	req := httptest.NewRequest(http.MethodGet, "/videos?page=3&limit=3", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Data       []json.RawMessage       `json:"data"`
		Pagination response.PaginationMeta `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding list body: %v", err)
	}

	if got := len(body.Data); got != 1 {
		t.Errorf("page 3 of 7 at limit 3 has %d items, want 1", got)
	}
	want := response.PaginationMeta{Total: 7, Page: 3, Limit: 3, TotalPages: 3, HasNext: false, HasPrevious: true}
	if body.Pagination != want {
		t.Errorf("pagination = %+v, want %+v", body.Pagination, want)
	}

	// Anonymous listings must have asked the repository for public videos
	// only; leaving Visibility nil would list every private video too.
	filter := repo.filterSeen()
	if filter.Visibility == nil || *filter.Visibility != domain.VisibilityPublic {
		t.Errorf("anonymous listing queried with visibility = %v, want public", filter.Visibility)
	}
}

// TestListVideosMine pins the mine=true switch: with a principal it filters by
// owner (surfacing that caller's private videos) instead of by visibility.
func TestListVideosMine(t *testing.T) {
	repo := newStubVideoRepo()
	me := principalFor(domain.RoleUser)
	seedVideo(repo, me.UserID, domain.VisibilityPublic)
	seedVideo(repo, me.UserID, domain.VisibilityPrivate)
	seedVideo(repo, uuid.New(), domain.VisibilityPublic) // someone else's

	router := videoRouter(newTestVideoHandler(repo), me)

	req := httptest.NewRequest(http.MethodGet, "/videos?mine=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Data       []json.RawMessage       `json:"data"`
		Pagination response.PaginationMeta `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding list body: %v", err)
	}
	if got := len(body.Data); got != 2 {
		t.Errorf("mine=true returned %d videos, want the caller's 2 (private included)", got)
	}
	if body.Pagination.Total != 2 {
		t.Errorf("total = %d, want 2", body.Pagination.Total)
	}

	filter := repo.filterSeen()
	if filter.OwnerID == nil || *filter.OwnerID != me.UserID {
		t.Errorf("mine=true queried with owner = %v, want the caller", filter.OwnerID)
	}
	if filter.Visibility != nil {
		t.Errorf("mine=true must not constrain visibility, got %v", *filter.Visibility)
	}
}

// TestDeleteVideoAuthorization pins the write-authorization ladder inside the
// handler: no caller is 401, a caller without ownership or delete_any_video is
// 403, and the owner or a moderator succeeds.
func TestDeleteVideoAuthorization(t *testing.T) {
	owner := appctx.Principal{UserID: uuid.New(), Username: "owner", Role: domain.RoleUser}

	tests := []struct {
		name        string
		caller      *appctx.Principal
		wantStatus  int
		wantCode    string
		wantDeleted bool
	}{
		{
			name:       "anonymous delete is 401",
			caller:     nil,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "non-owner without delete_any_video is 403",
			caller:     principalFor(domain.RoleUser),
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:        "owner may delete",
			caller:      &owner,
			wantStatus:  http.StatusOK,
			wantDeleted: true,
		},
		{
			name:        "moderator may delete anyone's video",
			caller:      principalFor(domain.RoleModerator),
			wantStatus:  http.StatusOK,
			wantDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newStubVideoRepo()
			video := seedVideo(repo, owner.UserID, domain.VisibilityPublic)
			router := videoRouter(newTestVideoHandler(repo), tt.caller)

			req := httptest.NewRequest(http.MethodDelete, "/videos/"+video.ID.String(), nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				env := decodeHandlerEnvelope(t, rec)
				if env.Error == nil || env.Error.Code != tt.wantCode {
					t.Errorf("error body = %s, want code %q", rec.Body.String(), tt.wantCode)
				}
			}
			if repo.has(video.ID) == tt.wantDeleted {
				t.Errorf("video present = %v after a %d response", repo.has(video.ID), rec.Code)
			}
		})
	}
}
