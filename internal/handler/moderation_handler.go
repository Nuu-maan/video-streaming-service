package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

// reviewActions are the outcomes ModerationService.ReviewReport understands.
// They are checked here so an unknown action is a 400 rather than a 500 raised
// deep in the service.
var reviewActions = map[string]bool{
	"delete_video": true,
	"ban_user":     true,
	"warn_user":    true,
	"dismiss":      true,
}

type ModerationHandler struct {
	moderation *service.ModerationService
	log        *logger.Logger
}

func NewModerationHandler(moderation *service.ModerationService, log *logger.Logger) *ModerationHandler {
	return &ModerationHandler{moderation: moderation, log: log}
}

type createReportRequest struct {
	VideoID     string `json:"video_id"`
	UserID      string `json:"user_id"`
	CommentID   string `json:"comment_id"`
	ReportType  string `json:"report_type" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description"`
}

type reviewReportRequest struct {
	Action string `json:"action" binding:"required"`
	Notes  string `json:"notes"`
}

type banUserRequest struct {
	Reason string `json:"reason" binding:"required"`
	// Duration is a Go duration such as "72h". Empty means a permanent ban.
	Duration string `json:"duration"`
}

// CreateReport files a report against a video, user, or comment. Reporting is a
// user action, so any authenticated caller may do it — not just moderators.
func (h *ModerationHandler) CreateReport(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required to report content")
		return
	}

	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "report_type and reason are required")
		return
	}

	report := &domain.ContentReport{
		ID:          uuid.New(),
		ReporterID:  principal.UserID,
		ReportType:  domain.ReportType(strings.TrimSpace(req.ReportType)),
		Reason:      strings.TrimSpace(req.Reason),
		Description: strings.TrimSpace(req.Description),
		Status:      domain.ReportStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	var bad bool
	report.VideoID, bad = optionalUUID(c, req.VideoID, "video_id")
	if bad {
		return
	}
	report.UserID, bad = optionalUUID(c, req.UserID, "user_id")
	if bad {
		return
	}
	report.CommentID, bad = optionalUUID(c, req.CommentID, "comment_id")
	if bad {
		return
	}

	if err := h.moderation.CreateReport(ctx, report); err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateReport):
			response.Error(c, http.StatusConflict, "DUPLICATE_REPORT", "You have already reported this content")
		case errors.Is(err, domain.ErrInvalidReportType), errors.Is(err, domain.ErrMissingReportTarget),
			errors.Is(err, domain.ErrMissingReportReason), errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to create report", err, map[string]interface{}{
				"reporter_id": principal.UserID,
			})
			response.InternalError(c, "Failed to create report")
		}
		return
	}

	response.Success(c, http.StatusCreated, report)
}

// ListPendingReports returns a page of reports awaiting moderator review.
func (h *ModerationHandler) ListPendingReports(c *gin.Context) {
	ctx := c.Request.Context()

	page := parsePage(c)

	reports, total, err := h.moderation.GetPendingReports(ctx, page.Limit, page.Offset)
	if err != nil {
		h.log.Error(ctx, "failed to list pending reports", err, nil)
		response.InternalError(c, "Failed to retrieve pending reports")
		return
	}

	response.SuccessWithList(c, reports, paginationMeta(int(total), page))
}

// ReviewReport resolves or dismisses a report, applying the moderator's chosen
// action to the reported content.
func (h *ModerationHandler) ReviewReport(c *gin.Context) {
	ctx := c.Request.Context()

	reportID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid report ID")
		return
	}

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	var req reviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "action is required")
		return
	}

	action := strings.TrimSpace(req.Action)
	if !reviewActions[action] {
		response.ValidationError(c, "action must be one of: delete_video, ban_user, warn_user, dismiss")
		return
	}

	// Banning through a report ends with a user losing their account, so it is
	// held to the same permission as the direct ban endpoint.
	if action == "ban_user" && !principal.HasPermission(domain.PermissionManageUsers) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Banning a user requires the manage_users permission")
		return
	}

	if err := h.moderation.ReviewReport(ctx, reportID, principal.UserID, action, strings.TrimSpace(req.Notes)); err != nil {
		switch {
		case errors.Is(err, domain.ErrReportNotFound):
			response.NotFound(c, "Report not found")
		case errors.Is(err, domain.ErrVideoNotFound):
			response.NotFound(c, "The reported video no longer exists")
		case errors.Is(err, domain.ErrUserNotFound):
			response.NotFound(c, "The reported user no longer exists")
		case errors.Is(err, domain.ErrMissingReportTarget), errors.Is(err, domain.ErrInvalidInput):
			response.ValidationError(c, err.Error())
		default:
			h.log.Error(ctx, "failed to review report", err, map[string]interface{}{
				"report_id": reportID,
				"action":    action,
			})
			response.InternalError(c, "Failed to review report")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":   "Report reviewed",
		"report_id": reportID,
		"action":    action,
	})
}

// BanUser bans a user, optionally for a fixed duration.
func (h *ModerationHandler) BanUser(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid user ID")
		return
	}

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	var req banUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "reason is required")
		return
	}

	// A moderator who bans themselves locks the account out with no way back in
	// through this API.
	if userID == principal.UserID {
		response.ValidationError(c, "You cannot ban yourself")
		return
	}

	var duration *time.Duration
	if raw := strings.TrimSpace(req.Duration); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			response.ValidationError(c, "duration must be a positive Go duration such as 72h")
			return
		}
		duration = &parsed
	}

	if err := h.moderation.BanUser(ctx, userID, principal.UserID, strings.TrimSpace(req.Reason), duration); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, "User not found")
			return
		}
		h.log.Error(ctx, "failed to ban user", err, map[string]interface{}{"user_id": userID})
		response.InternalError(c, "Failed to ban user")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "User banned",
		"user_id": userID,
	})
}

// UnbanUser lifts a ban.
func (h *ModerationHandler) UnbanUser(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid user ID")
		return
	}

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	if err := h.moderation.UnbanUser(ctx, userID, principal.UserID); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.NotFound(c, "User not found")
			return
		}
		h.log.Error(ctx, "failed to unban user", err, map[string]interface{}{"user_id": userID})
		response.InternalError(c, "Failed to unban user")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "User unbanned",
		"user_id": userID,
	})
}

// optionalUUID parses an optional report target. It returns nil for an absent
// target, and reports true when the value was present but malformed — in which
// case the error response has already been written.
func optionalUUID(c *gin.Context, raw, field string) (*uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	id, err := validator.ValidateUUID(raw)
	if err != nil {
		response.ValidationError(c, "Invalid "+field)
		return nil, true
	}

	return &id, false
}
