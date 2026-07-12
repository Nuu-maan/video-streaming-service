package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
)

const (
	testSecret      = "test-secret-key-that-is-long-enough"
	testOtherSecret = "a-completely-different-secret-key!!"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestTokens(secret string) *jwt.TokenService {
	return jwt.NewTokenService(secret, time.Hour, "test-issuer")
}

// mintToken returns a signed token for a caller with the given role.
func mintToken(t *testing.T, tokens *jwt.TokenService, userID uuid.UUID, username string, role domain.Role) string {
	t.Helper()
	token, err := tokens.GenerateToken(userID.String(), username, string(role))
	if err != nil {
		t.Fatalf("GenerateToken() unexpected error: %v", err)
	}
	return token
}

func TestRequireAuth(t *testing.T) {
	tokens := newTestTokens(testSecret)
	auth := NewAuthenticator(tokens)

	userID := uuid.New()
	validToken := mintToken(t, tokens, userID, "gopher", domain.RoleUser)
	foreignToken := mintToken(t, newTestTokens(testOtherSecret), userID, "gopher", domain.RoleUser)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{
			name:       "valid bearer token",
			header:     "Bearer " + validToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing authorization header",
			header:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed header without bearer prefix",
			header:     validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong scheme",
			header:     "Basic " + validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bearer prefix with empty token",
			header:     "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "garbage token",
			header:     "Bearer not-a-jwt-at-all",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token signed with a different secret",
			header:     "Bearer " + foreignToken,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				handlerRan     bool
				gotPrincipal   appctx.Principal
				principalFound bool
			)

			router := gin.New()
			router.GET("/protected", auth.RequireAuth(), func(c *gin.Context) {
				handlerRan = true
				gotPrincipal, principalFound = appctx.PrincipalFrom(c.Request.Context())
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus != http.StatusOK {
				if handlerRan {
					t.Error("the protected handler ran despite the request being rejected")
				}
				return
			}

			if !principalFound {
				t.Fatal("PrincipalFrom() reported no principal on an authenticated request")
			}
			if gotPrincipal.UserID != userID {
				t.Errorf("principal UserID = %v, want %v", gotPrincipal.UserID, userID)
			}
			if gotPrincipal.Username != "gopher" {
				t.Errorf("principal Username = %q, want %q", gotPrincipal.Username, "gopher")
			}
			if gotPrincipal.Role != domain.RoleUser {
				t.Errorf("principal Role = %q, want %q", gotPrincipal.Role, domain.RoleUser)
			}
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	tokens := newTestTokens(testSecret)
	auth := NewAuthenticator(tokens)

	userID := uuid.New()
	validToken := mintToken(t, tokens, userID, "gopher", domain.RoleUser)

	tests := []struct {
		name          string
		header        string
		wantPrincipal bool
	}{
		{
			name:          "no token, request still served anonymously",
			header:        "",
			wantPrincipal: false,
		},
		{
			name:          "valid token attaches the principal",
			header:        "Bearer " + validToken,
			wantPrincipal: true,
		},
		{
			// A public endpoint must not fail closed on a bad token; the caller is
			// simply anonymous.
			name:          "garbage token is treated as anonymous",
			header:        "Bearer garbage",
			wantPrincipal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				handlerRan     bool
				gotPrincipal   appctx.Principal
				principalFound bool
			)

			router := gin.New()
			router.GET("/public", auth.OptionalAuth(), func(c *gin.Context) {
				handlerRan = true
				gotPrincipal, principalFound = appctx.PrincipalFrom(c.Request.Context())
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/public", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
			}
			if !handlerRan {
				t.Fatal("the handler did not run; OptionalAuth must never block a request")
			}
			if principalFound != tt.wantPrincipal {
				t.Fatalf("principal present = %v, want %v", principalFound, tt.wantPrincipal)
			}
			if tt.wantPrincipal && gotPrincipal.UserID != userID {
				t.Errorf("principal UserID = %v, want %v", gotPrincipal.UserID, userID)
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	tokens := newTestTokens(testSecret)
	auth := NewAuthenticator(tokens)

	tests := []struct {
		name       string
		role       domain.Role
		permission domain.Permission
		wantStatus int
	}{
		{
			name:       "user cannot delete any video",
			role:       domain.RoleUser,
			permission: domain.PermissionDeleteAnyVideo,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can delete any video",
			role:       domain.RoleAdmin,
			permission: domain.PermissionDeleteAnyVideo,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user cannot manage users",
			role:       domain.RoleUser,
			permission: domain.PermissionManageUsers,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can manage users",
			role:       domain.RoleAdmin,
			permission: domain.PermissionManageUsers,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user can upload a video",
			role:       domain.RoleUser,
			permission: domain.PermissionUploadVideo,
			wantStatus: http.StatusOK,
		},
		{
			name:       "moderator cannot manage users",
			role:       domain.RoleModerator,
			permission: domain.PermissionManageUsers,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerRan bool

			router := gin.New()
			router.DELETE("/admin/videos/:id",
				auth.RequireAuth(),
				auth.RequirePermission(tt.permission),
				func(c *gin.Context) {
					handlerRan = true
					c.Status(http.StatusOK)
				},
			)

			token := mintToken(t, tokens, uuid.New(), "caller", tt.role)
			req := httptest.NewRequest(http.MethodDelete, "/admin/videos/abc", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if handlerRan != (tt.wantStatus == http.StatusOK) {
				t.Errorf("handler ran = %v, want %v", handlerRan, tt.wantStatus == http.StatusOK)
			}
		})
	}
}

// TestRequirePermissionWithoutAuth covers RequirePermission mounted without
// RequireAuth in front of it: with no principal in the context it must reject
// with 401 rather than admit the caller.
func TestRequirePermissionWithoutAuth(t *testing.T) {
	auth := NewAuthenticator(newTestTokens(testSecret))

	handlerRan := false
	router := gin.New()
	router.GET("/admin", auth.RequirePermission(domain.PermissionManageUsers), func(c *gin.Context) {
		handlerRan = true
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if handlerRan {
		t.Error("the protected handler ran without an authenticated principal")
	}
}
