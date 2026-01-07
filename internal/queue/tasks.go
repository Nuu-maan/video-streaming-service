package queue

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeVideoProcessing     = "video:process"
	TypeThumbnailGeneration = "video:thumbnail"
	TypeCleanup             = "video:cleanup"
)

type VideoProcessingPayload struct {
	VideoID   string   `json:"video_id"`
	Qualities []string `json:"qualities"`
	Priority  int      `json:"priority"`
}

type ThumbnailGenerationPayload struct {
	VideoID string `json:"video_id"`
}

type CleanupPayload struct {
	VideoID string `json:"video_id"`
	Paths   []string `json:"paths"`
}

func NewVideoProcessingTask(payload VideoProcessingPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal video processing payload: %w", err)
	}
	return asynq.NewTask(TypeVideoProcessing, payloadBytes), nil
}

func ParseVideoProcessingPayload(task *asynq.Task) (*VideoProcessingPayload, error) {
	var payload VideoProcessingPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal video processing payload: %w", err)
	}
	return &payload, nil
}

func NewThumbnailGenerationTask(payload ThumbnailGenerationPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal thumbnail generation payload: %w", err)
	}
	return asynq.NewTask(TypeThumbnailGeneration, payloadBytes), nil
}

func ParseThumbnailGenerationPayload(task *asynq.Task) (*ThumbnailGenerationPayload, error) {
	var payload ThumbnailGenerationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal thumbnail generation payload: %w", err)
	}
	return &payload, nil
}

func NewCleanupTask(payload CleanupPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cleanup payload: %w", err)
	}
	return asynq.NewTask(TypeCleanup, payloadBytes), nil
}

func ParseCleanupPayload(task *asynq.Task) (*CleanupPayload, error) {
	var payload CleanupPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cleanup payload: %w", err)
	}
	return &payload, nil
}
