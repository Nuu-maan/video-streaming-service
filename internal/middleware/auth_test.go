package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

const (
	testSecret      = "test-secret-key-that-is-long-enough"
	testOtherSecret = "a-completely-different-secret-key!!"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestTokens(secret string) *jwt.TokenService {
	return jwt.NewTokenService(secret, time.Hour, 24*time.Hour, "test-issuer")
}

func testLogger() *logger.Logger {
	return logger.New("production", "error")
}

// newTestAuthenticator builds an Authenticator without revocation checking,
// for the tests that only exercise signature validation.
func newTestAuthenticator(tokens *jwt.TokenService) *Authenticator {
	return NewAuthenticator(tokens, nil, false, testLogger())
}

// fakeRevocations is an in-memory RevocationChecker. A non-nil err simulates
// the revocation store being unreachable.
type fakeRevocations struct {
	revokedJTIs map[string]bool
	minIssuedAt map[string]time.Time
	err         error
}

func (f *fakeRevocations) IsRevoked(_ context.Context, jti, userID string, issuedAt time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.revokedJTIs[jti] {
		return true, nil
	}
	if cutoff, ok := f.minIssuedAt[userID]; ok && issuedAt.Before(cutoff) {
		return true, nil
	}
	return false, nil
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
	auth := newTestAuthenticator(tokens)

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
	auth := newTestAuthenticator(tokens)

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
	auth := newTestAuthenticator(tokens)

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

func TestRequireAuthRevocation(t *testing.T) {
	tokens := newTestTokens(testSecret)
	userID := uuid.New()
	token := mintToken(t, tokens, userID, "gopher", domain.RoleUser)

	claims, err := tokens.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.ID == "" {
		t.Fatal("token has no jti; revocation tests cannot key on it")
	}

	tests := []struct {
		name        string
		revocations RevocationChecker
		failOpen    bool
		wantStatus  int
	}{
		{
			name:        "token not revoked",
			revocations: &fakeRevocations{},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "token revoked by jti",
			revocations: &fakeRevocations{revokedJTIs: map[string]bool{claims.ID: true}},
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name: "all sessions revoked after issuance",
			revocations: &fakeRevocations{minIssuedAt: map[string]time.Time{
				userID.String(): time.Now().Add(time.Minute),
			}},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "store unreachable fails closed by default",
			revocations: &fakeRevocations{err: errors.New("redis: connection refused")},
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name:        "store unreachable with fail-open configured",
			revocations: &fakeRevocations{err: errors.New("redis: connection refused")},
			failOpen:    true,
			wantStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthenticator(tokens, tt.revocations, tt.failOpen, testLogger())

			var handlerRan bool
			router := gin.New()
			router.GET("/protected", auth.RequireAuth(), func(c *gin.Context) {
				handlerRan = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

// TestOptionalAuthRevokedToken verifies a revoked token on a public endpoint
// downgrades to anonymous rather than failing the request.
func TestOptionalAuthRevokedToken(t *testing.T) {
	tokens := newTestTokens(testSecret)
	token := mintToken(t, tokens, uuid.New(), "gopher", domain.RoleUser)

	claims, err := tokens.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	auth := NewAuthenticator(tokens, &fakeRevocations{
		revokedJTIs: map[string]bool{claims.ID: true},
	}, false, testLogger())

	var principalFound bool
	router := gin.New()
	router.GET("/public", auth.OptionalAuth(), func(c *gin.Context) {
		_, principalFound = appctx.PrincipalFrom(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if principalFound {
		t.Error("a revoked token attached a principal on a public endpoint")
	}
}

// TestRequirePermissionWithoutAuth covers RequirePermission mounted without
// RequireAuth in front of it: with no principal in the context it must reject
// with 401 rather than admit the caller.
func TestRequirePermissionWithoutAuth(t *testing.T) {
	auth := newTestAuthenticator(newTestTokens(testSecret))

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
