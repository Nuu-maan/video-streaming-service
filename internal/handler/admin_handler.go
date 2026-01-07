package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/queue"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/pkg/logger"
	"github.com/orchids/video-streaming/pkg/response"
	"github.com/orchids/video-streaming/pkg/validator"
)

type AdminHandler struct {
	videoRepo   repository.VideoRepository
	queueClient *queue.QueueClient
	inspector   *asynq.Inspector
	log         *logger.Logger
}

func NewAdminHandler(
	videoRepo repository.VideoRepository,
	queueClient *queue.QueueClient,
	redisAddr string,
	log *logger.Logger,
) *AdminHandler {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
	return &AdminHandler{
		videoRepo:   videoRepo,
		queueClient: queueClient,
		inspector:   inspector,
		log:         log,
	}
}

func (h *AdminHandler) RetryVideo(c *gin.Context) {
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

	if video.Status != domain.VideoStatusFailed {
		response.BadRequest(c, "Only failed videos can be retried")
		return
	}

	if err := h.videoRepo.UpdateStatus(ctx, videoID, domain.VideoStatusUploading); err != nil {
		h.log.Error(ctx, "failed to update video status", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to update video status")
		return
	}

	if err := h.queueClient.EnqueueVideoProcessing(ctx, videoID.String(), 1); err != nil {
		h.log.Error(ctx, "failed to enqueue video processing", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to enqueue video for processing")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":  "Video processing retry initiated",
		"video_id": videoID,
	})
}

func (h *AdminHandler) GetQueueStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.inspector.GetQueueInfo("default")
	if err != nil {
		h.log.Error(ctx, "failed to get queue stats", map[string]interface{}{
			"error": err.Error(),
		})
		response.InternalError(c, "Failed to retrieve queue statistics")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"active":      stats.Active,
		"pending":     stats.Pending,
		"scheduled":   stats.Scheduled,
		"retry":       stats.Retry,
		"archived":    stats.Archived,
		"completed":   stats.Completed,
		"aggregating": stats.Aggregating,
		"processed":   stats.Processed,
		"failed":      stats.Failed,
		"paused":      stats.Paused,
		"size":        stats.Size,
	})
}

func (h *AdminHandler) ListActiveWorkers(c *gin.Context) {
	ctx := c.Request.Context()

	workers, err := h.inspector.ListServers()
	if err != nil {
		h.log.Error(ctx, "failed to list workers", map[string]interface{}{
			"error": err.Error(),
		})
		response.InternalError(c, "Failed to retrieve worker information")
		return
	}

	workerInfo := make([]gin.H, 0, len(workers))
	for _, worker := range workers {
		workerInfo = append(workerInfo, gin.H{
			"host":         worker.Host,
			"pid":          worker.PID,
			"server_id":    worker.ID,
			"concurrency":  worker.Concurrency,
			"queues":       worker.Queues,
			"started":      worker.Started,
			"active_tasks": worker.ActiveWorkers,
		})
	}

	response.Success(c, http.StatusOK, gin.H{
		"workers": workerInfo,
		"count":   len(workerInfo),
	})
}
