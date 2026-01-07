package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/pkg/logger"
	"github.com/orchids/video-streaming/pkg/validator"
	"github.com/orchids/video-streaming/web/templates"
)

type PageHandler struct {
	videoRepo repository.VideoRepository
	log       *logger.Logger
}

func NewPageHandler(
	videoRepo repository.VideoRepository,
	log *logger.Logger,
) *PageHandler {
	return &PageHandler{
		videoRepo: videoRepo,
		log:       log,
	}
}

func (h *PageHandler) UploadPage(c *gin.Context) {
	component := templates.UploadPage()
	component.Render(c.Request.Context(), c.Writer)
}

func (h *PageHandler) VideoListPage(c *gin.Context) {
	ctx := c.Request.Context()

	videos, err := h.videoRepo.List(ctx, 50, 0)
	if err != nil {
		h.log.Error(ctx, "failed to list videos for page", map[string]interface{}{
			"error": err.Error(),
		})
		c.String(500, "Failed to load videos")
		return
	}

	component := templates.VideoListPage(videos)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *PageHandler) VideoPlayerPage(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	videoID, err := validator.ValidateUUID(idParam)
	if err != nil {
		c.String(400, "Invalid video ID")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			c.String(404, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to get video for page", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		c.String(500, "Failed to load video")
		return
	}

	component := templates.VideoPlayerPage(video)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *PageHandler) UploadSuccessPartial(c *gin.Context, video *domain.Video) {
	component := templates.UploadSuccess(video)
	component.Render(c.Request.Context(), c.Writer)
}
