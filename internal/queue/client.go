package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/orchids/video-streaming/pkg/logger"
)

type QueueClient struct {
	client *asynq.Client
	logger *logger.Logger
}

func NewQueueClient(redisAddr string, logger *logger.Logger) *QueueClient {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &QueueClient{
		client: client,
		logger: logger,
	}
}

func (q *QueueClient) Close() error {
	return q.client.Close()
}

func (q *QueueClient) EnqueueVideoProcessing(ctx context.Context, videoID string, priority int) error {
	payload := VideoProcessingPayload{
		VideoID:   videoID,
		Qualities: []string{"360p", "480p", "720p", "1080p"},
		Priority:  priority,
	}

	task, err := NewVideoProcessingTask(payload)
	if err != nil {
		q.logger.Error(ctx, "failed to create video processing task", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		return fmt.Errorf("failed to create task: %w", err)
	}

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Timeout(1 * time.Hour),
		asynq.Queue(getQueueName(priority)),
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		q.logger.Error(ctx, "failed to enqueue video processing task", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Info(ctx, "video processing task enqueued", map[string]interface{}{
		"video_id": videoID,
		"task_id":  info.ID,
		"queue":    info.Queue,
	})

	return nil
}

func (q *QueueClient) EnqueueThumbnailGeneration(ctx context.Context, videoID string) error {
	payload := ThumbnailGenerationPayload{
		VideoID: videoID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeThumbnailGeneration, payloadBytes)
	opts := []asynq.Option{
		asynq.MaxRetry(2),
		asynq.Timeout(5 * time.Minute),
		asynq.Queue("default"),
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		q.logger.Error(ctx, "failed to enqueue thumbnail generation task", map[string]interface{}{
			"error":    err.Error(),
			"video_id": videoID,
		})
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Info(ctx, "thumbnail generation task enqueued", map[string]interface{}{
		"video_id": videoID,
		"task_id":  info.ID,
	})

	return nil
}

func getQueueName(priority int) string {
	if priority >= 2 {
		return "critical"
	} else if priority <= -1 {
		return "low"
	}
	return "default"
}
