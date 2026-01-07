package queue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/orchids/video-streaming/internal/service"
	"github.com/orchids/video-streaming/pkg/logger"
)

type VideoProcessingHandler struct {
	transcodingService *service.TranscodingService
	logger             *logger.Logger
}

func NewVideoProcessingHandler(transcodingService *service.TranscodingService, logger *logger.Logger) *VideoProcessingHandler {
	return &VideoProcessingHandler{
		transcodingService: transcodingService,
		logger:             logger,
	}
}

func (h *VideoProcessingHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := ParseVideoProcessingPayload(task)
	if err != nil {
		h.logger.Error(ctx, "failed to parse video processing payload", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("parse payload: %w", err)
	}

	h.logger.Info(ctx, "processing video task", map[string]interface{}{
		"video_id":  payload.VideoID,
		"qualities": payload.Qualities,
		"priority":  payload.Priority,
		"task_id":   task.ResultWriter().TaskID(),
	})

	if err := h.transcodingService.ProcessVideo(ctx, payload.VideoID); err != nil {
		h.logger.Error(ctx, "video processing failed", map[string]interface{}{
			"video_id": payload.VideoID,
			"error":    err.Error(),
			"task_id":  task.ResultWriter().TaskID(),
		})
		return fmt.Errorf("process video: %w", err)
	}

	h.logger.Info(ctx, "video processing completed", map[string]interface{}{
		"video_id": payload.VideoID,
		"task_id":  task.ResultWriter().TaskID(),
	})

	return nil
}

type ThumbnailGenerationHandler struct {
	logger *logger.Logger
}

func NewThumbnailGenerationHandler(logger *logger.Logger) *ThumbnailGenerationHandler {
	return &ThumbnailGenerationHandler{
		logger: logger,
	}
}

func (h *ThumbnailGenerationHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := ParseThumbnailGenerationPayload(task)
	if err != nil {
		h.logger.Error(ctx, "failed to parse thumbnail generation payload", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("parse payload: %w", err)
	}

	h.logger.Info(ctx, "generating thumbnail", map[string]interface{}{
		"video_id": payload.VideoID,
	})

	return nil
}
