package app

// Integration tests that drive the full HTTP stack — router, middleware, and
// real handlers — through httptest, with the repositories and the object store
// replaced by in-memory fakes. CI has no Postgres and no Redis, so everything
// here must hold without either: what needs a database goes through the
// repository interfaces, and the one Redis-backed dependency in the request
// path (the playlist cache) is given a client pointed at a closed port, which
// exercises the same fall-through-to-storage path a Redis outage would.
//
// Each test names the regression it pins. None of them skip when
// infrastructure is absent, because none of them need any.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Nuu-maan/video-streaming-service/internal/cache"
	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/handler"
	"github.com/Nuu-maan/video-streaming-service/internal/middleware"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/internal/storage"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

const (
	integrationSecret = "integration-test-secret-key-0123456789"
	testOrigin        = "https://frontend.example"
)

// ---------------------------------------------------------------------------
// In-memory fakes for the persistence interfaces
// ---------------------------------------------------------------------------

type memVideoRepo struct {
	mu     sync.Mutex
	videos map[uuid.UUID]*domain.Video
}

func newMemVideoRepo() *memVideoRepo {
	return &memVideoRepo{videos: make(map[uuid.UUID]*domain.Video)}
}

func (r *memVideoRepo) Create(_ context.Context, v *domain.Video) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.videos[v.ID] = v
	return nil
}

func (r *memVideoRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.videos[id]
	if !ok {
		return nil, domain.ErrVideoNotFound
	}
	return v, nil
}

func (r *memVideoRepo) matching(filter repository.VideoFilter) []*domain.Video {
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
		if filter.Search != "" &&
			!strings.Contains(strings.ToLower(v.Title+" "+v.Description), strings.ToLower(filter.Search)) {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func (r *memVideoRepo) List(_ context.Context, filter repository.VideoFilter, page repository.Page) ([]*domain.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
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

func (r *memVideoRepo) Count(_ context.Context, filter repository.VideoFilter) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.matching(filter)), nil
}

func (r *memVideoRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.videos[id]; !ok {
		return domain.ErrVideoNotFound
	}
	delete(r.videos, id)
	return nil
}

func (r *memVideoRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.VideoStatus) error {
	return nil
}
func (r *memVideoRepo) UpdateProgress(_ context.Context, _ uuid.UUID, _ int) error      { return nil }
func (r *memVideoRepo) UpdateDuration(_ context.Context, _ uuid.UUID, _ int) error      { return nil }
func (r *memVideoRepo) UpdateResolution(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *memVideoRepo) UpdateHLSInfo(_ context.Context, _ uuid.UUID, _ string, _ bool) error {
	return nil
}
func (r *memVideoRepo) MarkAsReady(_ context.Context, _ uuid.UUID, _ []string, _ string) error {
	return nil
}
func (r *memVideoRepo) MarkAsFailed(_ context.Context, _ uuid.UUID) error { return nil }

type memUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (r *memUserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Username == u.Username || existing.Email == u.Email {
			return domain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return nil
}

func (r *memUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *memUserRepo) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
	return nil
}

func (r *memUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id)
	return nil
}

func (r *memUserRepo) List(_ context.Context, _ repository.Page) ([]*domain.User, error) {
	return nil, nil
}

func (r *memUserRepo) Count(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users), nil
}

func (r *memUserRepo) BanUser(_ context.Context, _ uuid.UUID, _ string, _ *time.Time) error {
	return nil
}
func (r *memUserRepo) UnbanUser(_ context.Context, _ uuid.UUID) error { return nil }

// memViewRepo fakes service.ViewTrackerRepository.
type memViewRepo struct {
	mu      sync.Mutex
	history []*domain.WatchHistory
}

func (r *memViewRepo) RecordView(_ context.Context, _, _ *uuid.UUID, _, _, _, _, _, _, _ string) error {
	return nil
}

func (r *memViewRepo) UpsertWatchHistory(_ context.Context, entry *domain.WatchHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.history {
		if existing.UserID == entry.UserID && existing.VideoID == entry.VideoID {
			r.history[i] = entry
			return nil
		}
	}
	r.history = append(r.history, entry)
	return nil
}

func (r *memViewRepo) ListWatchHistory(_ context.Context, userID uuid.UUID, _, _ int) ([]*domain.WatchHistory, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.WatchHistory
	for _, e := range r.history {
		if e.UserID == userID {
			out = append(out, e)
		}
	}
	return out, len(out), nil
}

func (r *memViewRepo) ClearWatchHistory(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.history[:0]
	for _, e := range r.history {
		if e.UserID != userID {
			kept = append(kept, e)
		}
	}
	r.history = kept
	return nil
}

func (r *memViewRepo) DeleteWatchHistoryEntry(_ context.Context, userID, videoID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.history {
		if e.UserID == userID && e.VideoID == videoID {
			r.history = append(r.history[:i], r.history[i+1:]...)
			return nil
		}
	}
	return domain.ErrWatchHistoryNotFound
}

func (r *memViewRepo) lastEntry() *domain.WatchHistory {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.history) == 0 {
		return nil
	}
	return r.history[len(r.history)-1]
}

// memStore fakes storage.Store with a map of key -> bytes.
type memStore struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMemStore() *memStore { return &memStore{files: make(map[string][]byte)} }

func (s *memStore) put(key string, content []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[key] = content
}

func (s *memStore) Save(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.put(key, content)
	return nil
}

type readSeekNopCloser struct{ *bytes.Reader }

func (readSeekNopCloser) Close() error { return nil }

func (s *memStore) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[key]
	if !ok {
		return nil, fmt.Errorf("no object %q", key)
	}
	return readSeekNopCloser{bytes.NewReader(content)}, nil
}

func (s *memStore) Stat(_ context.Context, key string) (storage.FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[key]
	if !ok {
		return storage.FileInfo{}, fmt.Errorf("no object %q", key)
	}
	return storage.FileInfo{Size: int64(len(content)), ModTime: time.Unix(1_700_000_000, 0)}, nil
}

func (s *memStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, key)
	return nil
}

func (s *memStore) DeletePrefix(_ context.Context, prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.files {
		if strings.HasPrefix(key, prefix) {
			delete(s.files, key)
		}
	}
	return nil
}

func (s *memStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.files[key]
	return ok, nil
}

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

type apiFixture struct {
	handler http.Handler
	tokens  *jwt.TokenService
	videos  *memVideoRepo
	users   *memUserRepo
	views   *memViewRepo
	store   *memStore
}

// newAPIFixture wires an App exactly as New does, but with the database-backed
// repositories replaced by in-memory fakes. Handlers this file never invokes
// stay nil, which is safe: registering a method value does not dereference its
// receiver, only calling it does.
func newAPIFixture(t *testing.T) *apiFixture {
	t.Helper()

	cfg := &config.Config{
		// production keeps gin in release mode, so the tests do not spew the
		// route table into the test log.
		Server: config.ServerConfig{Environment: "production"},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{testOrigin},
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			MaxAge:         time.Hour,
		},
		RateLimit: config.RateLimitConfig{Enabled: false},
		Auth: config.AuthConfig{
			JWTSecret:       integrationSecret,
			JWTIssuer:       "integration-test",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}

	log := logger.New("production", "error")
	tokens := jwt.NewTokenService(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL, cfg.Auth.JWTIssuer)

	videos := newMemVideoRepo()
	users := newMemUserRepo()
	views := &memViewRepo{}
	store := newMemStore()

	// CI has no Redis. The playlist cache gets a client aimed at a port nothing
	// listens on, with retries disabled so each call fails immediately; the
	// streaming handler then falls through to the store, which is exactly the
	// degradation a real Redis outage produces.
	deadRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 100 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() { _ = deadRedis.Close() })

	cacheSvc := cache.NewCacheService(deadRedis, 128)
	t.Cleanup(cacheSvc.Close)

	uploadSvc := service.NewUploadService(videos, service.NewFFmpegService(log), &cfg.Storage, store, log)

	// sessions (the Redis-backed revocation store) is nil. That is deliberate
	// and safe for what this file tests: every token rejection asserted here
	// happens at signature/type validation, before revocation is ever
	// consulted. The successful redemption of a refresh token is NOT covered
	// here because it requires the revocation store — see the coverage notes.
	authSvc := service.NewAuthService(users, tokens, nil, cfg.Auth, log)

	// The view tracker's Redis client is nil: only SaveProgress (which never
	// touches Redis) and the pre-tracker validation paths of RecordView are
	// driven through it.
	tracker := service.NewViewTracker(views, nil, log)

	a := &App{
		cfg:              cfg,
		log:              log,
		startedAt:        time.Now(),
		authenticator:    middleware.NewAuthenticator(tokens, nil, false, log),
		authHandler:      handler.NewAuthHandler(authSvc, users, log),
		videoHandler:     handler.NewVideoHandler(uploadSvc, videos, nil, log, cfg),
		streamingHandler: handler.NewStreamingHandler(videos, cacheSvc, store, log),
		viewHandler:      handler.NewViewHandler(tracker, log),
	}

	return &apiFixture{
		handler: a.Handler(),
		tokens:  tokens,
		videos:  videos,
		users:   users,
		views:   views,
		store:   store,
	}
}

// seedUser stores a user and returns it with a valid access token.
func (f *apiFixture) seedUser(t *testing.T, username string, role domain.Role) (*domain.User, string) {
	t.Helper()
	user := &domain.User{
		ID:       uuid.New(),
		Username: username,
		Email:    username + "@example.com",
		Role:     role,
	}
	if err := f.users.Create(nil, user); err != nil {
		t.Fatalf("seeding user %q: %v", username, err)
	}
	token, err := f.tokens.GenerateToken(user.ID.String(), user.Username, string(user.Role))
	if err != nil {
		t.Fatalf("minting access token: %v", err)
	}
	return user, token
}

// seedPlayableVideo stores a fully-transcoded video and the store objects the
// streaming routes serve, so an authorized request succeeds end to end.
func (f *apiFixture) seedPlayableVideo(t *testing.T, owner uuid.UUID, visibility domain.VideoVisibility) *domain.Video {
	t.Helper()

	id := uuid.New()
	ownerID := owner
	masterKey := "transcoded/" + id.String() + "/hls/master.m3u8"
	thumbKey := "thumbnails/" + id.String() + ".jpg"

	video := &domain.Video{
		ID:                 id,
		UserID:             &ownerID,
		Title:              "video-" + id.String()[:8],
		Filename:           "source.mp4",
		FileSize:           1 << 20,
		MimeType:           "video/mp4",
		Status:             domain.VideoStatusReady,
		Visibility:         visibility,
		AvailableQualities: []string{"720p"},
		HLSReady:           true,
		HLSMasterPath:      &masterKey,
		ThumbnailPath:      &thumbKey,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := f.videos.Create(nil, video); err != nil {
		t.Fatalf("seeding video: %v", err)
	}

	prefix := "transcoded/" + id.String()
	f.store.put(prefix+"/hls/master.m3u8", []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000000\n720p/playlist.m3u8\n"))
	f.store.put(prefix+"/hls/720p/playlist.m3u8", []byte("#EXTM3U\n#EXTINF:4.0,\nsegment_000.ts\n#EXT-X-ENDLIST\n"))
	f.store.put(prefix+"/hls/720p/segment_000.ts", []byte("fake-mpegts-bytes"))
	f.store.put(prefix+"/720p.mp4", []byte("fake-mp4-bytes"))
	f.store.put(thumbKey, []byte("fake-jpeg-bytes"))

	return video
}

// request performs one HTTP request against the full router. body == "" sends
// no body; otherwise it is sent as JSON. token == "" leaves the request
// anonymous.
func (f *apiFixture) request(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

// envelope mirrors the API's JSON response envelope for assertions.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Pagination *response.PaginationMeta `json:"pagination"`
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response is not the JSON envelope: %v (body: %s)", err, rec.Body.String())
	}
	return env
}

// errorCode asserts the response is the error envelope and returns its code.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	env := decodeEnvelope(t, rec)
	if env.Success {
		t.Fatalf("expected an error envelope, got success=true (body: %s)", rec.Body.String())
	}
	if env.Error == nil {
		t.Fatalf("error envelope has no error object (body: %s)", rec.Body.String())
	}
	return env.Error.Code
}

// ---------------------------------------------------------------------------
// 1. Private videos answer 404, never 403
// ---------------------------------------------------------------------------

// TestPrivateVideoIs404NotForbidden pins the private-video access rule on the
// full router: a private video must return 404 NOT_FOUND — never 403 — to
// anyone who is not the owner (or a watch_private holder), on the metadata
// routes AND on every media route.
//
// Regression: PermissionWatchPrivate used to be checked nowhere, so private
// videos were world-readable; and a 403 here would confirm to a prober that
// the video exists, which is exactly what the 404 is for.
func TestPrivateVideoIs404NotForbidden(t *testing.T) {
	f := newAPIFixture(t)

	owner, ownerToken := f.seedUser(t, "owner", domain.RoleUser)
	_, otherToken := f.seedUser(t, "other", domain.RoleUser)
	_, premiumToken := f.seedUser(t, "premium", domain.RolePremium)

	video := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPrivate)
	base := "/api/v1/videos/" + video.ID.String()

	subPaths := []string{
		"", // GET /videos/:id
		"/status",
		"/hls/master.m3u8",
		"/hls/720p/playlist.m3u8",
		"/hls/720p/segment_000.ts",
		"/stream/720p",
		"/thumbnail",
	}

	callers := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "anonymous", token: "", wantStatus: http.StatusNotFound},
		{name: "authenticated non-owner", token: otherToken, wantStatus: http.StatusNotFound},
		{name: "owner", token: ownerToken, wantStatus: http.StatusOK},
		// premium holds watch_private, the one permission that opens private
		// videos to a non-owner.
		{name: "premium with watch_private", token: premiumToken, wantStatus: http.StatusOK},
	}

	for _, caller := range callers {
		for _, sub := range subPaths {
			t.Run(caller.name+" GET "+base+sub, func(t *testing.T) {
				rec := f.request(t, http.MethodGet, base+sub, caller.token, "")

				if rec.Code == http.StatusForbidden {
					t.Fatalf("a private video answered 403; denial must be 404 so it does not confirm existence (body: %s)", rec.Body.String())
				}
				if rec.Code != caller.wantStatus {
					t.Fatalf("status = %d, want %d (body: %s)", rec.Code, caller.wantStatus, rec.Body.String())
				}
				if caller.wantStatus == http.StatusNotFound {
					if code := errorCode(t, rec); code != "NOT_FOUND" {
						t.Errorf("error code = %q, want %q", code, "NOT_FOUND")
					}
				}
			})
		}
	}

	// The denial and the success must differ only by who is asking: the owner
	// gets the real playlist bytes, raw, not wrapped in the JSON envelope.
	t.Run("owner receives the raw playlist", func(t *testing.T) {
		rec := f.request(t, http.MethodGet, base+"/hls/master.m3u8", ownerToken, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		if !strings.HasPrefix(rec.Body.String(), "#EXTM3U") {
			t.Errorf("playlist body does not start with #EXTM3U: %q", rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "mpegurl") {
			t.Errorf("Content-Type = %q, want an m3u8 type", ct)
		}
	})

	// The metadata response must expose the computed URLs and withhold the
	// server-side storage keys.
	t.Run("owner metadata carries hls_url, not storage paths", func(t *testing.T) {
		rec := f.request(t, http.MethodGet, base, ownerToken, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"hls_url"`) || !strings.Contains(body, `"thumbnail_url"`) {
			t.Errorf("video JSON is missing hls_url/thumbnail_url: %s", body)
		}
		for _, leaked := range []string{"hls_master_path", "thumbnail_path", "file_path"} {
			if strings.Contains(body, leaked) {
				t.Errorf("video JSON leaks server-side storage key %q: %s", leaked, body)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// 2. Write authorization: 401 without a caller, 403 without the permission
// ---------------------------------------------------------------------------

// TestWriteAuthorizationMatrix pins that an unauthenticated write is 401 and
// an authenticated-but-unpermitted write is 403, at both enforcement layers:
// route middleware (upload, admin) and in-handler ownership checks (delete).
//
// Regression: upload used to be anonymous (uploads had no owner) and the admin
// surface used to be completely unauthenticated.
func TestWriteAuthorizationMatrix(t *testing.T) {
	f := newAPIFixture(t)

	owner, ownerToken := f.seedUser(t, "uploader", domain.RoleUser)
	_, userToken := f.seedUser(t, "user", domain.RoleUser)
	_, guestToken := f.seedUser(t, "guest", domain.RoleGuest)
	_, modToken := f.seedUser(t, "mod", domain.RoleModerator)

	tests := []struct {
		name       string
		method     string
		path       func() string
		token      string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "anonymous upload is 401",
			method:     http.MethodPost,
			path:       func() string { return "/api/v1/videos/upload" },
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "guest without upload_video is 403",
			method:     http.MethodPost,
			path:       func() string { return "/api/v1/videos/upload" },
			token:      guestToken,
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:   "anonymous delete is 401",
			method: http.MethodDelete,
			path: func() string {
				v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)
				return "/api/v1/videos/" + v.ID.String()
			},
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:   "non-owner delete without delete_any_video is 403",
			method: http.MethodDelete,
			path: func() string {
				v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)
				return "/api/v1/videos/" + v.ID.String()
			},
			token:      userToken,
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:   "owner delete is allowed",
			method: http.MethodDelete,
			path: func() string {
				v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)
				return "/api/v1/videos/" + v.ID.String()
			},
			token:      ownerToken,
			wantStatus: http.StatusOK,
		},
		{
			name:   "moderator with delete_any_video may delete another user's video",
			method: http.MethodDelete,
			path: func() string {
				v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)
				return "/api/v1/videos/" + v.ID.String()
			},
			token:      modToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "anonymous admin read is 401",
			method:     http.MethodGet,
			path:       func() string { return "/api/v1/admin/queue/stats" },
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "user without moderate_content cannot read admin queue stats",
			method:     http.MethodGet,
			path:       func() string { return "/api/v1/admin/queue/stats" },
			token:      userToken,
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:       "moderator without manage_users cannot ban",
			method:     http.MethodPost,
			path:       func() string { return "/api/v1/admin/users/" + uuid.NewString() + "/ban" },
			token:      modToken,
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.request(t, tt.method, tt.path(), tt.token, "")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				if code := errorCode(t, rec); code != tt.wantCode {
					t.Errorf("error code = %q, want %q", code, tt.wantCode)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. Pagination reports the real total
// ---------------------------------------------------------------------------

// TestListVideosPaginationTotal pins that the paginated envelope's total comes
// from a Count over the same filter as the page, not from len(page).
//
// Regression: the handler used to report len(currentPage) as the total, so
// total_pages was always 1 and has_next was always false — a frontend could
// never render page two.
func TestListVideosPaginationTotal(t *testing.T) {
	f := newAPIFixture(t)
	owner, _ := f.seedUser(t, "creator", domain.RoleUser)

	// 45 public videos and 5 private ones. The private ones must appear in
	// neither the page nor the total of an anonymous listing.
	for i := 0; i < 45; i++ {
		v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)
		v.Title = fmt.Sprintf("public-%02d", i)
	}
	for i := 0; i < 5; i++ {
		v := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPrivate)
		v.Title = fmt.Sprintf("private-%02d", i)
	}

	rec := f.request(t, http.MethodGet, "/api/v1/videos?page=2&limit=20", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Success    bool                    `json:"success"`
		Data       []json.RawMessage       `json:"data"`
		Pagination response.PaginationMeta `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding list envelope: %v (body: %s)", err, rec.Body.String())
	}

	if !body.Success {
		t.Error("success = false on a 200 listing")
	}
	if got := len(body.Data); got != 20 {
		t.Errorf("page size = %d, want 20", got)
	}
	want := response.PaginationMeta{Total: 45, Page: 2, Limit: 20, TotalPages: 3, HasNext: true, HasPrevious: true}
	if body.Pagination != want {
		t.Errorf("pagination = %+v, want %+v", body.Pagination, want)
	}

	// The last page carries the remainder and has_next flips off.
	rec = f.request(t, http.MethodGet, "/api/v1/videos?page=3&limit=20", "", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding list envelope: %v", err)
	}
	if got := len(body.Data); got != 5 {
		t.Errorf("last page size = %d, want 5", got)
	}
	if body.Pagination.HasNext {
		t.Error("has_next = true on the final page")
	}
	if !body.Pagination.HasPrevious {
		t.Error("has_previous = false on page 3")
	}
}

// ---------------------------------------------------------------------------
// 4. CORS exposes the headers a cross-origin player needs
// ---------------------------------------------------------------------------

// TestCORSExposesStreamingHeaders pins the CORS grant that makes the API
// consumable from a separate frontend origin at all: without Content-Range and
// Accept-Ranges in Access-Control-Expose-Headers, a cross-origin player cannot
// observe the result of its own Range requests, so seeking in hls.js and the
// MP4 fallback silently break — the whole point of a standalone API server.
func TestCORSExposesStreamingHeaders(t *testing.T) {
	f := newAPIFixture(t)
	owner, _ := f.seedUser(t, "creator", domain.RoleUser)
	video := f.seedPlayableVideo(t, owner.ID, domain.VisibilityPublic)

	assertCORSGrant := func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, testOrigin)
		}
		exposed := rec.Header().Get("Access-Control-Expose-Headers")
		for _, name := range []string{"Content-Range", "Accept-Ranges", "Content-Length", "X-Request-ID"} {
			if !strings.Contains(exposed, name) {
				t.Errorf("Access-Control-Expose-Headers = %q, missing %q", exposed, name)
			}
		}
	}

	t.Run("preflight on a media route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/videos/"+video.ID.String()+"/stream/720p", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "range")
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		assertCORSGrant(t, rec)

		// Range must be allowed as a request header even though the fixture's
		// config does not list it, or a player cannot send Range cross-origin.
		allowed := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
		for _, name := range []string{"range", "authorization", "content-type"} {
			if !strings.Contains(allowed, name) {
				t.Errorf("Access-Control-Allow-Headers = %q, missing %q", allowed, name)
			}
		}
	})

	t.Run("actual media response carries the grant", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/videos/"+video.ID.String()+"/hls/master.m3u8", nil)
		req.Header.Set("Origin", testOrigin)
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		assertCORSGrant(t, rec)
	})

	t.Run("range request on the MP4 fallback is honoured with 206", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/videos/"+video.ID.String()+"/stream/720p", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Range", "bytes=0-3")
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusPartialContent, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Range"); !strings.HasPrefix(got, "bytes 0-3/") {
			t.Errorf("Content-Range = %q, want a bytes 0-3/... range", got)
		}
		assertCORSGrant(t, rec)
	})
}

// ---------------------------------------------------------------------------
// 5. Access and refresh tokens are not interchangeable
// ---------------------------------------------------------------------------

// TestTokenTypesAreNotInterchangeable pins the access/refresh split: a refresh
// token presented as API credentials is 401, and an access token presented at
// /auth/refresh is 401.
//
// Regression: the two used to be the same thing — refresh accepted the
// caller's access token (making refresh useless once it expired), and a
// long-lived refresh token worked as an everyday API credential (turning a
// captured one into a week-long key).
func TestTokenTypesAreNotInterchangeable(t *testing.T) {
	f := newAPIFixture(t)
	user, accessToken := f.seedUser(t, "session", domain.RoleUser)

	refreshToken, err := f.tokens.GenerateRefreshToken(user.ID.String(), user.Username, string(user.Role))
	if err != nil {
		t.Fatalf("minting refresh token: %v", err)
	}

	t.Run("access token works as API credentials", func(t *testing.T) {
		rec := f.request(t, http.MethodGet, "/api/v1/auth/me", accessToken, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("refresh token is rejected as API credentials", func(t *testing.T) {
		for _, path := range []string{"/api/v1/auth/me", "/api/v1/me/history"} {
			rec := f.request(t, http.MethodGet, path, refreshToken, "")
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("GET %s with a refresh token = %d, want 401 (body: %s)", path, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("refresh token cannot authorize a write", func(t *testing.T) {
		video := f.seedPlayableVideo(t, user.ID, domain.VisibilityPublic)
		rec := f.request(t, http.MethodPost, "/api/v1/videos/"+video.ID.String()+"/progress",
			refreshToken, `{"position": 10, "duration": 100}`)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("access token is rejected at /auth/refresh", func(t *testing.T) {
		rec := f.request(t, http.MethodPost, "/api/v1/auth/refresh", "",
			fmt.Sprintf(`{"refresh_token": %q}`, accessToken))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
		}
		if code := errorCode(t, rec); code != "UNAUTHORIZED" {
			t.Errorf("error code = %q, want %q", code, "UNAUTHORIZED")
		}
	})

	t.Run("garbage at /auth/refresh is 401", func(t *testing.T) {
		rec := f.request(t, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token": "not-a-jwt"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing refresh_token is a validation error", func(t *testing.T) {
		rec := f.request(t, http.MethodPost, "/api/v1/auth/refresh", "", `{}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
		}
		if code := errorCode(t, rec); code != "VALIDATION_ERROR" {
			t.Errorf("error code = %q, want %q", code, "VALIDATION_ERROR")
		}
	})
}

// ---------------------------------------------------------------------------
// 6. Watch progress bounds
// ---------------------------------------------------------------------------

// TestSaveProgressBounds pins the resume-position contract: position and
// duration are non-negative seconds and position must lie within duration,
// anything else is a 400 — never silently clamped or stored.
func TestSaveProgressBounds(t *testing.T) {
	f := newAPIFixture(t)
	user, token := f.seedUser(t, "viewer", domain.RoleUser)
	video := f.seedPlayableVideo(t, user.ID, domain.VisibilityPublic)
	path := "/api/v1/videos/" + video.ID.String() + "/progress"

	tests := []struct {
		name       string
		token      string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "valid progress is saved",
			token:      token,
			body:       `{"position": 50, "duration": 100, "completed": false}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "position beyond duration is rejected",
			token:      token,
			body:       `{"position": 150, "duration": 100}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "negative position is rejected",
			token:      token,
			body:       `{"position": -1, "duration": 100}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "non-JSON body is rejected",
			token:      token,
			body:       `position=50`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "anonymous progress is 401",
			token:      "",
			body:       `{"position": 50, "duration": 100}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.request(t, http.MethodPost, path, tt.token, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				if code := errorCode(t, rec); code != tt.wantCode {
					t.Errorf("error code = %q, want %q", code, tt.wantCode)
				}
			}
		})
	}

	// Exactly one entry — from the one valid request — may have been stored.
	entry := f.views.lastEntry()
	if entry == nil {
		t.Fatal("the valid progress request stored nothing")
	}
	if entry.LastPosition != 50 || entry.UserID != user.ID || entry.VideoID != video.ID {
		t.Errorf("stored entry = %+v, want position 50 for the caller and video", entry)
	}
}

// ---------------------------------------------------------------------------
// 7. The error envelope
// ---------------------------------------------------------------------------

// TestValidationErrorEnvelope pins the error envelope's exact shape:
// {"success": false, "error": {"code": "...", "message": "..."}} with no data
// key. Frontends branch on error.code, so the shape is a contract.
func TestValidationErrorEnvelope(t *testing.T) {
	f := newAPIFixture(t)

	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCode string
	}{
		{
			name:     "login with no fields",
			method:   http.MethodPost,
			path:     "/api/v1/auth/login",
			body:     `{}`,
			wantCode: "VALIDATION_ERROR",
		},
		{
			name:     "login with a non-JSON body",
			method:   http.MethodPost,
			path:     "/api/v1/auth/login",
			body:     `identifier=alice&password=x`,
			wantCode: "VALIDATION_ERROR",
		},
		{
			name:     "video read with a malformed UUID",
			method:   http.MethodGet,
			path:     "/api/v1/videos/not-a-uuid",
			body:     "",
			wantCode: "VALIDATION_ERROR",
		},
		{
			// The view endpoint's anonymous contract: without a session_id an
			// anonymous view cannot be deduplicated, so it is refused.
			name:     "anonymous view without session_id",
			method:   http.MethodPost,
			path:     "/api/v1/videos/" + uuid.NewString() + "/view",
			body:     `{}`,
			wantCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.request(t, tt.method, tt.path, "", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
			}

			// Decode into a raw map so absent keys are distinguishable from
			// zero values: the envelope must carry no data key on an error.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("error response is not JSON: %v (body: %s)", err, rec.Body.String())
			}
			if string(raw["success"]) != "false" {
				t.Errorf("success = %s, want false", raw["success"])
			}
			if _, hasData := raw["data"]; hasData {
				t.Errorf("error envelope carries a data key: %s", rec.Body.String())
			}

			env := decodeEnvelope(t, rec)
			if env.Error == nil {
				t.Fatalf("no error object in envelope: %s", rec.Body.String())
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("error code = %q, want %q", env.Error.Code, tt.wantCode)
			}
			if env.Error.Message == "" {
				t.Error("error message is empty")
			}
		})
	}
}
