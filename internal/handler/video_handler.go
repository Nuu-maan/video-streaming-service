package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/queue"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

type VideoHandler struct {
	uploadService *service.UploadService
	videoRepo     repository.VideoRepository
	queueClient   *queue.QueueClient
	log           *logger.Logger
	cfg           *config.Config
}

func NewVideoHandler(
	uploadService *service.UploadService,
	videoRepo repository.VideoRepository,
	queueClient *queue.QueueClient,
	log *logger.Logger,
	cfg *config.Config,
) *VideoHandler {
	return &VideoHandler{
		uploadService: uploadService,
		videoRepo:     videoRepo,
		queueClient:   queueClient,
		log:           log,
		cfg:           cfg,
	}
}

// Upload accepts a multipart video upload and queues it for transcoding. It
// requires an authenticated caller: the uploaded video is recorded against them
// so ownership can later be enforced on delete.
func (h *VideoHandler) Upload(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required to upload")
		return
	}

	file, header, err := c.Request.FormFile("video")
	if err != nil {
		response.ValidationError(c, "A video file is required")
		return
	}
	defer file.Close()

	video, err := h.uploadService.UploadVideo(ctx, service.UploadRequest{
		File:        file,
		Header:      header,
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		OwnerID:     principal.UserID,
		Visibility:  domain.VideoVisibility(c.PostForm("visibility")),
	})
	if err != nil {
		h.respondUploadError(c, err, header.Filename)
		return
	}

	// The video is safely recorded; failing to queue it is recoverable via the
	// admin retry endpoint, so report success rather than losing the upload.
	if err := h.queueClient.EnqueueVideoProcessing(ctx, video.ID.String(), 0); err != nil {
		h.log.Error(ctx, "video stored but could not be queued for processing", err, map[string]interface{}{
			"video_id": video.ID,
		})
	}

	response.Success(c, http.StatusCreated, video)
}

// respondUploadError maps upload failures onto status codes. Client mistakes
// (too large, wrong format, bad title) must not be reported as 500s.
func (h *VideoHandler) respondUploadError(c *gin.Context, err error, filename string) {
	ctx := c.Request.Context()

	switch {
	case errors.Is(err, validator.ErrFileTooLarge), errors.Is(err, domain.ErrFileSizeTooLarge):
		response.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, validator.ErrInvalidFormat), errors.Is(err, domain.ErrInvalidFormat),
		errors.Is(err, validator.ErrCorruptVideo):
		response.Error(c, http.StatusUnsupportedMediaType, "INVALID_FORMAT", err.Error())
	case errors.Is(err, validator.ErrInvalidTitle), errors.Is(err, domain.ErrInvalidTitle),
		errors.Is(err, domain.ErrTitleTooLong), errors.Is(err, domain.ErrInvalidInput):
		response.ValidationError(c, err.Error())
	default:
		h.log.Error(ctx, "upload failed", err, map[string]interface{}{"filename": filename})
		response.InternalError(c, "Failed to upload video")
	}
}

// ListVideos returns a page of videos. Anonymous callers see only public,
// ready videos; authenticated callers additionally see their own.
func (h *VideoHandler) ListVideos(c *gin.Context) {
	ctx := c.Request.Context()

	page := parsePage(c)
	filter := repository.VideoFilter{Search: strings.TrimSpace(c.Query("search"))}

	if status := domain.VideoStatus(strings.TrimSpace(c.Query("status"))); status != "" {
		filter.Status = &status
	}

	// Without an explicit visibility filter, every private video in the table
	// would be listed to anonymous callers.
	if principal, ok := appctx.PrincipalFrom(ctx); ok && c.Query("mine") == "true" {
		owner := principal.UserID
		filter.OwnerID = &owner
	} else {
		visibility := domain.VisibilityPublic
		filter.Visibility = &visibility
	}

	videos, err := h.videoRepo.List(ctx, filter, page)
	if err != nil {
		h.log.Error(ctx, "failed to list videos", err, nil)
		response.InternalError(c, "Failed to retrieve videos")
		return
	}

	total, err := h.videoRepo.Count(ctx, filter)
	if err != nil {
		h.log.Error(ctx, "failed to count videos", err, nil)
		response.InternalError(c, "Failed to retrieve videos")
		return
	}

	response.SuccessWithList(c, videos, paginationMeta(total, page))
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	video, ok := h.loadVideo(c)
	if !ok {
		return
	}
	if !canViewVideo(c.Request.Context(), video) {
		response.NotFound(c, "Video not found")
		return
	}
	response.Success(c, http.StatusOK, video)
}

// canViewVideo reports whether the request behind ctx may read video. Only
// private restricts anything — unlisted means "reachable by link" — and the
// rule follows the role model: the owner, plus anyone granted watch_private.
// Listings already exclude private videos; without this check on direct reads
// and on streaming, "private" would be indistinguishable from "unlisted".
// Callers respond with 404, never 403, so denial does not confirm existence.
func canViewVideo(ctx context.Context, video *domain.Video) bool {
	if video.Visibility != domain.VisibilityPrivate {
		return true
	}
	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		return false
	}
	return video.IsOwnedBy(principal.UserID) || principal.HasPermission(domain.PermissionWatchPrivate)
}

// DeleteVideo removes a video. Only its owner, or a user holding
// PermissionDeleteAnyVideo, may do so.
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	ctx := c.Request.Context()

	video, ok := h.loadVideo(c)
	if !ok {
		return
	}

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	if !video.IsOwnedBy(principal.UserID) && !principal.HasPermission(domain.PermissionDeleteAnyVideo) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "You may only delete your own videos")
		return
	}

	if err := h.videoRepo.Delete(ctx, video.ID); err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to delete video", err, map[string]interface{}{"video_id": video.ID})
		response.InternalError(c, "Failed to delete video")
		return
	}

	h.uploadService.RemoveVideoFiles(ctx, video)

	response.Success(c, http.StatusOK, gin.H{"message": "Video deleted"})
}

// GetVideoStatus reports transcoding progress for a video.
func (h *VideoHandler) GetVideoStatus(c *gin.Context) {
	video, ok := h.loadVideo(c)
	if !ok {
		return
	}
	if !canViewVideo(c.Request.Context(), video) {
		response.NotFound(c, "Video not found")
		return
	}

	status := gin.H{
		"id":                  video.ID,
		"status":              video.Status,
		"progress":            video.TranscodingProgress,
		"available_qualities": video.AvailableQualities,
		"message":             statusMessage(video.Status),
	}
	if video.ThumbnailPath != nil {
		status["thumbnail"] = *video.ThumbnailPath
	}

	response.Success(c, http.StatusOK, status)
}

// loadVideo resolves the :id path parameter to a video, writing the error
// response itself and reporting false when it could not. Every read handler
// repeated this block.
func (h *VideoHandler) loadVideo(c *gin.Context) (*domain.Video, bool) {
	ctx := c.Request.Context()

	videoID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return nil, false
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return nil, false
		}
		h.log.Error(ctx, "failed to load video", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to retrieve video")
		return nil, false
	}

	return video, true
}

func statusMessage(status domain.VideoStatus) string {
	switch status {
	case domain.VideoStatusProcessing:
		return "Video is being processed"
	case domain.VideoStatusReady:
		return "Video is ready to stream"
	case domain.VideoStatusFailed:
		return "Video processing failed"
	default:
		return "Video is queued for processing"
	}
}

// parsePage reads page/limit query parameters, clamping them to a sane window
// so a caller cannot request an unbounded result set.
func parsePage(c *gin.Context) repository.Page {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	if err != nil || limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	return repository.Page{Limit: limit, Offset: (page - 1) * limit}
}

// paginationMeta derives the response envelope from the true row count. The
// handler previously passed len(currentPage) as the total, so TotalPages was
// always 1 and HasNext was always false.
func paginationMeta(total int, page repository.Page) response.PaginationMeta {
	currentPage := page.Offset/page.Limit + 1
	totalPages := (total + page.Limit - 1) / page.Limit

	return response.PaginationMeta{
		Total:       total,
		Page:        currentPage,
		Limit:       page.Limit,
		TotalPages:  totalPages,
		HasNext:     currentPage < totalPages,
		HasPrevious: currentPage > 1,
	}
}
