package handler

import (
	"errors"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
	"github.com/Nuu-maan/video-streaming-service/web/templates"
	"github.com/gin-gonic/gin"
)

// publicVisibility is the filter value for anonymous browsing. The rendered
// pages are unauthenticated, so they must never list private videos.
var publicVisibility = domain.VisibilityPublic

// pageListLimit caps how many videos the browse page renders at once.
const pageListLimit = 50

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

	videos, err := h.videoRepo.List(ctx, repository.VideoFilter{Visibility: &publicVisibility}, repository.Page{Limit: pageListLimit})
	if err != nil {
		h.log.Error(ctx, "failed to list videos for page", err, map[string]interface{}{})
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
		h.log.Error(ctx, "failed to get video for page", err, map[string]interface{}{
			"video_id": videoID,
		})
		c.String(500, "Failed to load video")
		return
	}

	// These pages are unauthenticated, so a private video reads as absent here
	// — same as the API answers an anonymous caller. Without this the page
	// leaks a private video's title and description to anyone with its ID.
	if video.Visibility == domain.VisibilityPrivate {
		c.String(404, "Video not found")
		return
	}

	component := templates.VideoPlayerPage(video)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *PageHandler) UploadSuccessPartial(c *gin.Context, video *domain.Video) {
	component := templates.UploadSuccess(video)
	component.Render(c.Request.Context(), c.Writer)
}
