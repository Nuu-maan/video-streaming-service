package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

// AccountHandler exposes email verification and password lifecycle endpoints.
type AccountHandler struct {
	emails *service.EmailService
	log    *logger.Logger
}

func NewAccountHandler(emails *service.EmailService, log *logger.Logger) *AccountHandler {
	return &AccountHandler{emails: emails, log: log}
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// SendVerificationEmail (re)sends a verification mail to the caller's own
// registered address. The address is never taken from the request body, so a
// caller cannot direct mail at an arbitrary inbox.
func (h *AccountHandler) SendVerificationEmail(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	if err := h.emails.SendVerificationEmail(ctx, principal.UserID); err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyVerified):
			response.Error(c, http.StatusConflict, "EMAIL_ALREADY_VERIFIED", "Your email address is already verified")
		case errors.Is(err, domain.ErrUserNotFound):
			response.NotFound(c, "User not found")
		default:
			h.log.Error(ctx, "failed to send verification email", err, nil)
			response.InternalError(c, "Failed to send verification email")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Verification email sent",
	})
}

// VerifyEmail consumes a verification token and marks the account verified.
func (h *AccountHandler) VerifyEmail(c *gin.Context) {
	ctx := c.Request.Context()

	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "token is required")
		return
	}

	if err := h.emails.VerifyEmail(ctx, req.Token); err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			response.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "The verification link is invalid or has already been used")
			return
		}
		h.log.Error(ctx, "failed to verify email", err, nil)
		response.InternalError(c, "Failed to verify email")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Email verified",
	})
}

// ForgotPassword starts a password reset. It answers with the same 200 body
// whether or not the address is registered — a distinguishable answer here
// would let anyone enumerate accounts — so even internal failures are logged
// and swallowed rather than surfaced.
func (h *AccountHandler) ForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()

	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "email is required")
		return
	}

	if err := h.emails.RequestPasswordReset(ctx, req.Email); err != nil {
		h.log.Error(ctx, "failed to process password reset request", err, nil)
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "If that email address is registered, a password reset link has been sent",
	})
}

// ResetPassword consumes a reset token and sets the new password.
func (h *AccountHandler) ResetPassword(c *gin.Context) {
	ctx := c.Request.Context()

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "token and password are required")
		return
	}

	if err := h.emails.ResetPassword(ctx, req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidToken):
			response.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "The reset link is invalid or has expired")
		case errors.Is(err, domain.ErrWeakPassword):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to reset password", err, nil)
			response.InternalError(c, "Failed to reset password")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Password has been reset. Please log in with your new password.",
	})
}

// ChangePassword sets a new password for the authenticated caller after
// verifying the current one.
func (h *AccountHandler) ChangePassword(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "current_password and new_password are required")
		return
	}

	if err := h.emails.ChangePassword(ctx, principal.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			response.Error(c, http.StatusBadRequest, "INVALID_CURRENT_PASSWORD", "Current password is incorrect")
		case errors.Is(err, domain.ErrWeakPassword):
			response.ValidationError(c, err.Error())
		case errors.Is(err, domain.ErrUserNotFound):
			response.NotFound(c, "User not found")
		default:
			h.log.Error(ctx, "failed to change password", err, nil)
			response.InternalError(c, "Failed to change password")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Password changed",
	})
}
