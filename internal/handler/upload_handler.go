package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/config"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/queue"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/internal/service"
	"github.com/orchids/video-streaming/pkg/logger"
	"github.com/orchids/video-streaming/pkg/response"
	"github.com/orchids/video-streaming/pkg/validator"
)

type UploadHandler struct {
	uploadService *service.UploadService
	videoRepo     repository.VideoRepository
	queueClient   *queue.QueueClient
	log           *logger.Logger
	config        *config.Config
}

func NewUploadHandler(
	uploadService *service.UploadService,
	videoRepo repository.VideoRepository,
	queueClient *queue.QueueClient,
	log *logger.Logger,
	config *config.Config,
) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
		videoRepo:     videoRepo,
		queueClient:   queueClient,
		log:           log,
		config:        config,
	}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	ctx := c.Request.Context()

	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		h.log.Error(ctx, "failed to parse multipart form", map[string]interface{}{
			"error": err.Error(),
		})
		response.BadRequest(c, "Invalid multipart form data")
		return
	}

	file, header, err := c.Request.FormFile("video")
	if err != nil {
		response.ValidationError(c, "Video file is required")
		return
	}
	defer file.Close()

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		response.ValidationError(c, "Title is required")
		return
	}

	description := strings.TrimSpace(c.PostForm("description"))

	video, err := h.uploadService.UploadVideo(ctx, file, header, title, description)
	if err != nil {
		h.log.Error(ctx, "upload failed", map[string]interface{}{
			"error":    err.Error(),
			"title":    title,
			"filename": header.Filename,
		})

		if errors.Is(err, validator.ErrFileTooLarge) {
			response.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", err.Error())
			return
		}
		if errors.Is(err, validator.ErrInvalidFormat) {
			response.Error(c, http.StatusBadRequest, "INVALID_FORMAT", err.Error())
			return
		}
		if errors.Is(err, validator.ErrInvalidTitle) {
			response.ValidationError(c, err.Error())
			return
		}

		response.InternalError(c, "Failed to upload video")
		return
	}

	if err := h.queueClient.EnqueueVideoProcessing(ctx, video.ID.String(), 0); err != nil {
		h.log.Error(ctx, "failed to enqueue video processing", map[string]interface{}{
			"error":    err.Error(),
			"video_id": video.ID,
		})
	}

	response.Success(c, http.StatusCreated, gin.H{
		"id":          video.ID,
		"title":       video.Title,
		"status":      video.Status,
		"file_size":   video.FileSize,
		"duration":    video.Duration,
		"resolution":  video.OriginalResolution,
		"created_at":  video.CreatedAt,
	})
}

func (h *UploadHandler) ListVideos(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	if err := validator.ValidatePageParams(page, limit); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	offset := (page - 1) * limit

	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))

	var videos []*domain.Video
	var err error

	if search != "" {
		videos, err = h.videoRepo.Search(ctx, search, limit, offset)
	} else if status != "" {
		videos, err = h.videoRepo.GetByStatus(ctx, domain.VideoStatus(status), limit, offset)
	} else {
		videos, err = h.videoRepo.List(ctx, limit, offset)
	}

	if err != nil {
		h.log.Error(ctx, "failed to list videos", map[string]interface{}{
			"error": err.Error(),
		})
		response.InternalError(c, "Failed to retrieve videos")
		return
	}

	total := len(videos)
	totalPages := (total + limit - 1) / limit

	meta := response.PaginationMeta{
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	response.SuccessWithList(c, videos, meta)
}

func (h *UploadHandler) GetVideo(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	videoID, err := validator.ValidateUUID(idParam)
	if err != nil {
		response.ValidationError(c, "Invalid video ID format")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to get video", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	response.Success(c, http.StatusOK, video)
}

func (h *UploadHandler) DeleteVideo(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	videoID, err := validator.ValidateUUID(idParam)
	if err != nil {
		response.ValidationError(c, "Invalid video ID format")
		return
	}

	if err := h.videoRepo.Delete(ctx, videoID); err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to delete video", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to delete video")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Video deleted successfully",
	})
}

func (h *UploadHandler) GetVideoStatus(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	videoID, err := validator.ValidateUUID(idParam)
	if err != nil {
		response.ValidationError(c, "Invalid video ID format")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to get video", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to retrieve video status")
		return
	}

	statusResponse := gin.H{
		"id":                  video.ID,
		"status":              video.Status,
		"progress":            video.TranscodingProgress,
		"available_qualities": video.AvailableQualities,
	}

	if video.Status == domain.VideoStatusProcessing {
		statusResponse["message"] = "Video is being processed..."
	} else if video.Status == domain.VideoStatusReady {
		statusResponse["message"] = "Video is ready to stream"
	} else if video.Status == domain.VideoStatusFailed {
		statusResponse["message"] = "Video processing failed"
	} else {
		statusResponse["message"] = "Video is queued for processing"
	}

	if video.ThumbnailPath != nil {
		statusResponse["thumbnail"] = *video.ThumbnailPath
	}

	response.Success(c, http.StatusOK, statusResponse)
}
