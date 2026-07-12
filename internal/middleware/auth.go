package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

const bearerPrefix = "Bearer "

// Authenticator validates bearer tokens and attaches the caller to the request
// context.
type Authenticator struct {
	tokens *jwt.TokenService
}

func NewAuthenticator(tokens *jwt.TokenService) *Authenticator {
	return &Authenticator{tokens: tokens}
}

// RequireAuth rejects requests that do not carry a valid bearer token.
func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, err := a.principalFromRequest(c)
		if err != nil {
			response.Unauthorized(c, "Valid authentication token required")
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(appctx.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

// OptionalAuth attaches the caller when a valid token is present but lets
// anonymous requests through. Use it on endpoints whose response varies by
// caller (listings, for instance) without requiring one.
//
// A malformed or expired token is treated as anonymous rather than rejected:
// the endpoint is public, and failing the request would make an expired token
// worse than no token at all.
func (a *Authenticator) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if principal, err := a.principalFromRequest(c); err == nil {
			c.Request = c.Request.WithContext(appctx.WithPrincipal(c.Request.Context(), principal))
		}
		c.Next()
	}
}

// RequirePermission rejects callers whose role does not grant permission. It
// must be mounted after RequireAuth.
func (a *Authenticator) RequirePermission(permission domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := appctx.PrincipalFrom(c.Request.Context())
		if !ok {
			response.Unauthorized(c, "Valid authentication token required")
			c.Abort()
			return
		}

		if !principal.HasPermission(permission) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to perform this action")
			c.Abort()
			return
		}

		c.Next()
	}
}

// principalFromRequest extracts and validates the bearer token.
func (a *Authenticator) principalFromRequest(c *gin.Context) (appctx.Principal, error) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return appctx.Principal{}, domain.ErrUnauthorized
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if token == "" {
		return appctx.Principal{}, domain.ErrUnauthorized
	}

	claims, err := a.tokens.ValidateToken(token)
	if err != nil {
		return appctx.Principal{}, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return appctx.Principal{}, domain.ErrInvalidToken
	}

	role := domain.Role(claims.Role)
	if !role.IsValid() {
		return appctx.Principal{}, domain.ErrInvalidToken
	}

	return appctx.Principal{
		UserID:   userID,
		Username: claims.Username,
		Role:     role,
	}, nil
}
