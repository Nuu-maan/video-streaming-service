package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/domain"
)

type VideoRepository interface {
	Create(ctx context.Context, video *domain.Video) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Video, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.VideoStatus) error
	UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error
	MarkAsReady(ctx context.Context, id uuid.UUID, qualities []string, thumbnailPath string) error
	MarkAsFailed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByStatus(ctx context.Context, status domain.VideoStatus, limit, offset int) ([]*domain.Video, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*domain.Video, error)
	UpdateDuration(ctx context.Context, id uuid.UUID, duration int) error
	UpdateResolution(ctx context.Context, id uuid.UUID, resolution string) error
}
