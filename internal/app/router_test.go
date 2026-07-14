package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/middleware"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// testApp builds an App with just enough wiring to register routes. The
// handler fields stay nil — taking a method value never dereferences its
// receiver — so these tests exercise route registration itself, which is
// where gin panics on a conflicting route table. That cannot be caught at
// compile time, and the table now contains static/param siblings
// (/videos/trending beside /videos/:id) and every route mounted twice.
func testApp() *App {
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "development"},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"https://frontend.example"},
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			MaxAge:         time.Hour,
		},
		RateLimit: config.RateLimitConfig{Enabled: false},
	}
	log := logger.New("development", "error")
	tokens := jwt.NewTokenService("test-secret", time.Minute, time.Hour, "test")

	return &App{
		cfg:           cfg,
		log:           log,
		startedAt:     time.Now(),
		authenticator: middleware.NewAuthenticator(tokens, nil, false, log),
	}
}

func TestHandlerRegistersVersionedAndAliasRoutes(t *testing.T) {
	router := testApp().Handler().(*gin.Engine)

	registered := make(map[string]bool)
	for _, r := range router.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	// One representative route per subsystem. Each must exist under the
	// canonical /api/v1 prefix and under the legacy /api alias, because the
	// alias is what keeps today's consumers working.
	wanted := []string{
		"POST /auth/login",
		"POST /auth/logout",
		"POST /auth/forgot-password",
		"GET /videos",
		"GET /videos/trending",
		"GET /videos/:id",
		"GET /videos/:id/related",
		"GET /videos/:id/hls/master.m3u8",
		"PUT /videos/:id/like",
		"POST /videos/:id/view",
		"GET /videos/:id/comments",
		"PATCH /comments/:id",
		"POST /users/:id/subscribe",
		"DELETE /playlists/:id/videos/:videoId",
		"GET /search",
		"GET /categories",
		"GET /me/feed",
		"DELETE /me/history/:videoId",
		"POST /me/notifications/read-all",
		"POST /me/notifications/:id/read",
		"POST /me/change-password",
		"POST /admin/users/:id/ban",
	}
	for _, want := range wanted {
		method, path, _ := strings.Cut(want, " ")
		for _, prefix := range []string{"/api", "/api/v1"} {
			if key := method + " " + prefix + path; !registered[key] {
				t.Errorf("route %s is not registered", key)
			}
		}
	}
}

// TestHandlerAnswersPreflight pins that OPTIONS is answered with the CORS
// grant on every path — matched or not — because browsers preflight routes
// that only register GET, and an unanswered preflight blocks the real call.
func TestHandlerAnswersPreflight(t *testing.T) {
	handler := testApp().Handler()

	paths := []string{
		"/api/v1/videos",
		"/api/v1/auth/login",
		"/api/videos/9f2b8e6c-4a53-4c25-9b0d-1f3a2b4c5d6e/hls/master.m3u8",
		"/api/v1/does-not-exist",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", "https://frontend.example")
		req.Header.Set("Access-Control-Request-Method", "GET")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("OPTIONS %s = %d, want %d", path, rec.Code, http.StatusNoContent)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example" {
			t.Errorf("OPTIONS %s Access-Control-Allow-Origin = %q, want the requesting origin", path, got)
		}
		if rec.Header().Get("Access-Control-Max-Age") == "" {
			t.Errorf("OPTIONS %s has no Access-Control-Max-Age", path)
		}
	}
}
