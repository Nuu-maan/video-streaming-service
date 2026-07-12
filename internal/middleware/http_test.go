package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name            string
		inboundID       string
		wantEchoedExact bool
	}{
		{
			name:            "generates an ID when none is supplied",
			inboundID:       "",
			wantEchoedExact: false,
		},
		{
			name:            "echoes an inbound ID unchanged",
			inboundID:       "inbound-correlation-id-123",
			wantEchoedExact: true,
		},
		{
			name:            "echoes an inbound UUID unchanged",
			inboundID:       "0d3f8a1e-4c2b-4f6d-9a1b-2c3d4e5f6a7b",
			wantEchoedExact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctxRequestID string

			router := gin.New()
			router.Use(RequestID())
			router.GET("/", func(c *gin.Context) {
				ctxRequestID = appctx.RequestID(c.Request.Context())
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.inboundID != "" {
				req.Header.Set(RequestIDHeader, tt.inboundID)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			got := rec.Header().Get(RequestIDHeader)
			if got == "" {
				t.Fatalf("response header %s is empty; a correlation ID must always be set", RequestIDHeader)
			}

			if tt.wantEchoedExact && got != tt.inboundID {
				t.Errorf("response header %s = %q, want the inbound ID %q echoed unchanged", RequestIDHeader, got, tt.inboundID)
			}

			if ctxRequestID != got {
				t.Errorf("request ID in context = %q, want it to match the response header %q", ctxRequestID, got)
			}
		})
	}
}

func TestRequestIDGeneratesDistinctIDs(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	seen := make(map[string]bool)
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		id := rec.Header().Get(RequestIDHeader)
		if id == "" {
			t.Fatal("generated request ID is empty")
		}
		if seen[id] {
			t.Fatalf("request ID %q was generated twice; IDs must be unique per request", id)
		}
		seen[id] = true
	}
}

func testCORSConfig() config.CORSConfig {
	return config.CORSConfig{
		AllowedOrigins: []string{"https://good.example"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Authorization"},
		MaxAge:         12 * time.Hour,
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantAllowOrigin string // "" means the header must be absent entirely
		wantCredentials bool
		wantVaryOrigin  bool
		wantHandlerRuns bool
	}{
		{
			name:            "allowed origin is echoed back with credentials",
			method:          http.MethodGet,
			origin:          "https://good.example",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "https://good.example",
			wantCredentials: true,
			wantVaryOrigin:  true,
			wantHandlerRuns: true,
		},
		{
			// An origin that is not on the list must get no CORS grant at all —
			// not a wildcard, not an echo.
			name:            "disallowed origin gets no allow-origin header",
			method:          http.MethodGet,
			origin:          "https://evil.example",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantCredentials: false,
			wantVaryOrigin:  false,
			wantHandlerRuns: true,
		},
		{
			name:            "origin that merely shares a suffix is not allowed",
			method:          http.MethodGet,
			origin:          "https://evil-good.example",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantCredentials: false,
			wantVaryOrigin:  false,
			wantHandlerRuns: true,
		},
		{
			name:            "same-origin request without an Origin header",
			method:          http.MethodGet,
			origin:          "",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
			wantCredentials: false,
			wantVaryOrigin:  false,
			wantHandlerRuns: true,
		},
		{
			name:            "preflight from an allowed origin returns 204",
			method:          http.MethodOptions,
			origin:          "https://good.example",
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "https://good.example",
			wantCredentials: true,
			wantVaryOrigin:  true,
			wantHandlerRuns: false,
		},
		{
			name:            "preflight from a disallowed origin returns 204 with no grant",
			method:          http.MethodOptions,
			origin:          "https://evil.example",
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "",
			wantCredentials: false,
			wantVaryOrigin:  false,
			wantHandlerRuns: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerRan bool

			router := gin.New()
			router.Use(CORS(testCORSConfig()))
			handler := func(c *gin.Context) {
				handlerRan = true
				c.Status(http.StatusOK)
			}
			router.GET("/resource", handler)
			router.OPTIONS("/resource", handler)

			req := httptest.NewRequest(tt.method, "/resource", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if handlerRan != tt.wantHandlerRuns {
				t.Errorf("handler ran = %v, want %v", handlerRan, tt.wantHandlerRuns)
			}

			header := rec.Header()

			if tt.wantAllowOrigin == "" {
				if _, present := header["Access-Control-Allow-Origin"]; present {
					t.Errorf("Access-Control-Allow-Origin = %q, want the header to be absent entirely",
						header.Get("Access-Control-Allow-Origin"))
				}
			} else if got := header.Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllowOrigin)
			}

			gotCredentials := header.Get("Access-Control-Allow-Credentials") == "true"
			if gotCredentials != tt.wantCredentials {
				t.Errorf("Access-Control-Allow-Credentials present = %v, want %v", gotCredentials, tt.wantCredentials)
			}

			gotVary := false
			for _, v := range header.Values("Vary") {
				if v == "Origin" {
					gotVary = true
				}
			}
			if gotVary != tt.wantVaryOrigin {
				t.Errorf("Vary: Origin present = %v, want %v (Vary: %v)", gotVary, tt.wantVaryOrigin, header.Values("Vary"))
			}

			if tt.wantAllowOrigin != "" {
				if got := header.Get("Access-Control-Allow-Methods"); got == "" {
					t.Error("Access-Control-Allow-Methods is missing on an allowed origin")
				}
				if got := header.Get("Access-Control-Allow-Headers"); got == "" {
					t.Error("Access-Control-Allow-Headers is missing on an allowed origin")
				}
				if got := header.Get("Access-Control-Max-Age"); got != "43200" {
					t.Errorf("Access-Control-Max-Age = %q, want %q", got, "43200")
				}
			}
		})
	}
}

// TestCORSWildcardDropsCredentials pins the documented behaviour: a wildcard
// allowlist must never be paired with Access-Control-Allow-Credentials, a
// combination browsers reject and which would be a CSRF hole if honoured.
func TestCORSWildcardDropsCredentials(t *testing.T) {
	cfg := testCORSConfig()
	cfg.AllowedOrigins = []string{"*"}

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://anything.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want it to be absent alongside a wildcard origin", got)
	}
}
