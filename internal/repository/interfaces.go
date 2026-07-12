// Package repository defines the persistence contracts the service layer
// depends on. Implementations live in subpackages (see repository/postgres).
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// VideoFilter narrows a video listing. The zero value matches every video.
type VideoFilter struct {
	// Status matches a single lifecycle status when set.
	Status *domain.VideoStatus
	// OwnerID restricts results to one uploader when set.
	OwnerID *uuid.UUID
	// Visibility restricts results to one visibility when set. Anonymous
	// listings must set this to VisibilityPublic; leaving it nil lists private
	// videos too.
	Visibility *domain.VideoVisibility
	// Search is a free-text query matched against title and description.
	Search string
}

// Page is a limit/offset window over a result set.
type Page struct {
	Limit  int
	Offset int
}

// VideoRepository persists videos.
//
// List and Count take the same filter, so a caller can page through results and
// still report an accurate total. The previous interface had no Count at all,
// which is why the handler reported len(currentPage) as the total and every
// listing claimed to be exactly one page long.
type VideoRepository interface {
	Create(ctx context.Context, video *domain.Video) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error)
	List(ctx context.Context, filter VideoFilter, page Page) ([]*domain.Video, error)
	Count(ctx context.Context, filter VideoFilter) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error

	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.VideoStatus) error
	UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error
	UpdateDuration(ctx context.Context, id uuid.UUID, duration int) error
	UpdateResolution(ctx context.Context, id uuid.UUID, resolution string) error
	UpdateHLSInfo(ctx context.Context, id uuid.UUID, hlsMasterPath string, hlsReady bool) error
	MarkAsReady(ctx context.Context, id uuid.UUID, qualities []string, thumbnailPath string) error
	MarkAsFailed(ctx context.Context, id uuid.UUID) error
}

// UserRepository persists users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page Page) ([]*domain.User, error)
	Count(ctx context.Context) (int, error)

	// BanUser bans id until the given time; a nil until means permanently.
	BanUser(ctx context.Context, id uuid.UUID, reason string, until *time.Time) error
	UnbanUser(ctx context.Context, id uuid.UUID) error
}
