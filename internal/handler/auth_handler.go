package handler

import (
	"errors"
	"net/http"

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
	Token string `json:"token" binding:"required"`
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

// Refresh exchanges a valid token for a fresh one.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "token is required")
		return
	}

	tokens, err := h.auth.Refresh(c.Request.Context(), req.Token)
	if err != nil {
		h.respondAuthError(c, err)
		return
	}

	response.Success(c, http.StatusOK, tokens)
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
	case errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrTokenExpired):
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
