package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

type AuthHandler struct {
	auth  *service.AuthService
	users repository.UserRepository
	log   *logger.Logger
}

func NewAuthHandler(auth *service.AuthService, users repository.UserRepository, log *logger.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, users: users, log: log}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	// Identifier is a username or an email address.
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type refreshRequest struct {
	// The field is named for the token it takes, and matches the name login
	// returns it under, so a client can hand back exactly what it was given.
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	// RefreshToken is optional: only clients that also hold a refresh token
	// send one so it can be revoked alongside the access token.
	RefreshToken string `json:"refresh_token"`
}

// Register creates an account and returns a token for it.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "username, a valid email, and password are required")
		return
	}

	tokens, err := h.auth.Register(c.Request.Context(), service.Registration{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.respondAuthError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, tokens)
}

// Login exchanges credentials for a token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "identifier and password are required")
		return
	}

	tokens, err := h.auth.Login(c.Request.Context(), service.Credentials{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {
		h.respondAuthError(c, err)
		return
	}

	response.Success(c, http.StatusOK, tokens)
}

// Refresh exchanges a refresh token for a new access token, so a session
// outlives the access token's few minutes without the user signing in again.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "refresh_token is required")
		return
	}

	tokens, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.respondAuthError(c, err)
		return
	}

	response.Success(c, http.StatusOK, tokens)
}

// Logout revokes the presented access token, and the refresh token too when
// the body carries one. The token stays cryptographically valid until expiry —
// revocation happens in the denylist the auth middleware consults, not in the
// token itself — so this is what actually ends the session server-side.
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	if _, ok := appctx.PrincipalFrom(ctx); !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	token := bearerToken(c)
	if token == "" {
		response.Unauthorized(c, "Authentication required")
		return
	}

	// The body is optional; a bare POST logs out the access token alone.
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.auth.Logout(ctx, token, strings.TrimSpace(req.RefreshToken)); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			response.Unauthorized(c, "Invalid or expired token")
			return
		}
		h.log.Error(ctx, "failed to log out", err, nil)
		response.InternalError(c, "Failed to log out")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Logged out"})
}

// LogoutAll revokes every outstanding session the caller has, on every device.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	if err := h.auth.RevokeAllSessions(ctx, principal.UserID); err != nil {
		h.log.Error(ctx, "failed to revoke all sessions", err, nil)
		response.InternalError(c, "Failed to log out")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "All sessions revoked"})
}

// bearerToken returns the raw token from the Authorization header, or "" when
// the header is absent or not a bearer credential.
func bearerToken(c *gin.Context) string {
	const prefix = "Bearer "
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// Me returns the authenticated caller's own account.
func (h *AuthHandler) Me(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.users.GetByID(ctx, principal.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, "User not found")
			return
		}
		h.log.Error(ctx, "failed to load current user", err, map[string]interface{}{
			"user_id": principal.UserID,
		})
		response.InternalError(c, "Failed to load user")
		return
	}

	response.Success(c, http.StatusOK, user)
}

// respondAuthError maps auth failures onto status codes.
//
// Bad credentials, a missing user, and a banned user must not be distinguishable
// by status code alone beyond what is intended: an unknown account and a wrong
// password both return 401 with the same message, so the endpoint cannot be used
// to enumerate registered usernames.
func (h *AuthHandler) respondAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUserNotFound):
		response.Unauthorized(c, "Invalid credentials")
	case errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrTokenExpired),
		// A token revoked by logout is a rejected credential, not a server fault:
		// without this it fell through to the 500 default, so signing out and then
		// refreshing reported an internal error instead of "sign in again".
		errors.Is(err, domain.ErrTokenRevoked):
		response.Unauthorized(c, "Invalid or expired token")
	case errors.Is(err, domain.ErrUserBanned):
		response.Error(c, http.StatusForbidden, "USER_BANNED", "This account is banned")
	case errors.Is(err, domain.ErrUserAlreadyExists):
		response.Error(c, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrWeakPassword),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		response.ValidationError(c, err.Error())
	default:
		h.log.Error(c.Request.Context(), "authentication failed", err, nil)
		response.InternalError(c, "Authentication failed")
	}
}
