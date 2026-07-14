package queue

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TypeVideoProcessing = "video:process"

type VideoProcessingPayload struct {
	VideoID   string   `json:"video_id"`
	Qualities []string `json:"qualities"`
	Priority  int      `json:"priority"`
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
