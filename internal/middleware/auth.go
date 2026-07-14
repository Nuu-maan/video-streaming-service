package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

const bearerPrefix = "Bearer "

// RevocationChecker answers whether an already-validated token has since been
// revoked. Satisfied by *service.SessionService; it is an interface here so
// the middleware does not depend on the service layer.
type RevocationChecker interface {
	IsRevoked(ctx context.Context, jti, userID string, issuedAt time.Time) (bool, error)
}

// Authenticator validates bearer tokens and attaches the caller to the request
// context.
type Authenticator struct {
	tokens *jwt.TokenService
	// revocations may be nil, which disables revocation checks entirely (used
	// by tests that only exercise signature validation).
	revocations RevocationChecker
	// revocationFailOpen accepts tokens when the revocation store is
	// unreachable instead of rejecting them. See checkRevocation for why the
	// default is to fail closed.
	revocationFailOpen bool
	log                *logger.Logger
}

func NewAuthenticator(tokens *jwt.TokenService, revocations RevocationChecker, revocationFailOpen bool, log *logger.Logger) *Authenticator {
	return &Authenticator{
		tokens:             tokens,
		revocations:        revocations,
		revocationFailOpen: revocationFailOpen,
		log:                log,
	}
}

// RequireAuth rejects requests that do not carry a valid bearer token.
func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, err := a.principalFromRequest(c)
		if err != nil {
			// When the revocation store is down and configured to fail closed,
			// the token may well be fine — 503 tells the client to retry
			// rather than to discard its credentials as a 401 would.
			if errors.Is(err, domain.ErrRevocationUnavailable) {
				response.Error(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "Authentication is temporarily unavailable")
			} else {
				response.Unauthorized(c, "Valid authentication token required")
			}
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
// A malformed, expired, or revoked token is treated as anonymous rather than
// rejected: the endpoint is public, and failing the request would make a bad
// token worse than no token at all. The same goes for an unreachable
// revocation store — anonymity grants nothing extra, so there is no reason to
// fail closed here.
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

	// Access tokens only. A refresh token is valid for days, so accepting one as
	// API credentials would hand out a long-lived key to anyone who captured it
	// — it is meant to be redeemable at exactly one endpoint and useless at every
	// other.
	claims, err := a.tokens.ValidateAccessToken(token)
	if err != nil {
		return appctx.Principal{}, domain.ErrInvalidToken
	}

	if err := a.checkRevocation(c.Request.Context(), claims); err != nil {
		return appctx.Principal{}, err
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

// checkRevocation rejects tokens that have been logged out or invalidated by a
// logout-all, at the cost of one Redis round trip per authenticated request —
// the unavoidable price of revocation actually working.
//
// When the store is unreachable the default is to FAIL CLOSED: a service that
// keeps honouring revoked tokens after a breach is strictly worse than one
// that is briefly down, because the operator revoking tokens is doing so
// precisely because those tokens are in the wrong hands. revocationFailOpen
// inverts that for deployments that prefer availability, and either way the
// failure is logged loudly so it cannot pass unnoticed.
func (a *Authenticator) checkRevocation(ctx context.Context, claims *jwt.Claims) error {
	if a.revocations == nil {
		return nil
	}

	// Millisecond precision, not the whole-second iat: the logout-all cutoff is
	// compared against this, and a second-granularity value lets a session minted
	// in the same second as the revocation slip through it.
	revoked, err := a.revocations.IsRevoked(ctx, claims.ID, claims.UserID, claims.IssuedAtTime())
	if err != nil {
		if a.log != nil {
			a.log.Error(ctx, "revocation store unreachable", err, map[string]interface{}{
				"fail_open": a.revocationFailOpen,
			})
		}
		if a.revocationFailOpen {
			return nil
		}
		return domain.ErrRevocationUnavailable
	}
	if revoked {
		return domain.ErrTokenRevoked
	}
	return nil
}
