package domain

import (
	"context"

	"github.com/google/uuid"
)

type InteractionType string

const (
	InteractionView     InteractionType = "view"
	InteractionLike     InteractionType = "like"
	InteractionDislike  InteractionType = "dislike"
	InteractionComment  InteractionType = "comment"
	InteractionShare    InteractionType = "share"
	InteractionSkip     InteractionType = "skip"
	InteractionComplete InteractionType = "complete"
)

type Interaction struct {
	UserID    uuid.UUID
	VideoID   uuid.UUID
	Type      InteractionType
	Weight    float64
	Timestamp int64
}

type RecommendationEngine interface {
	GetPersonalizedRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]*Video, error)
	GetSimilarVideos(ctx context.Context, videoID uuid.UUID, limit int) ([]*Video, error)
	GetTrendingVideos(ctx context.Context, category string, limit int) ([]*Video, error)
	RecordInteraction(ctx context.Context, interaction *Interaction) error
}

func (it InteractionType) Weight() float64 {
	switch it {
	case InteractionView:
		return 1.0
	case InteractionLike:
		return 5.0
	case InteractionDislike:
		return -3.0
	case InteractionComment:
		return 3.0
	case InteractionShare:
		return 10.0
	case InteractionSkip:
		return -2.0
	case InteractionComplete:
		return 2.0
	default:
		return 0.0
	}
}

func (it InteractionType) Validate() error {
	validTypes := map[InteractionType]bool{
		InteractionView:     true,
		InteractionLike:     true,
		InteractionDislike:  true,
		InteractionComment:  true,
		InteractionShare:    true,
		InteractionSkip:     true,
		InteractionComplete: true,
	}

	if !validTypes[it] {
		return ErrInvalidInput
	}

	return nil
}

// TrendingWindow is the span over which trending engagement is counted. The
// values are the literal query-string tokens the API accepts.
type TrendingWindow string

const (
	TrendingWindow24h TrendingWindow = "24h"
	TrendingWindow7d  TrendingWindow = "7d"
	TrendingWindow30d TrendingWindow = "30d"
)

// Hours returns the window length in whole hours, the unit the repository
// hands to Postgres' make_interval.
func (w TrendingWindow) Hours() int {
	switch w {
	case TrendingWindow7d:
		return 7 * 24
	case TrendingWindow30d:
		return 30 * 24
	default:
		return 24
	}
}

func (w TrendingWindow) Validate() error {
	switch w {
	case TrendingWindow24h, TrendingWindow7d, TrendingWindow30d:
		return nil
	default:
		return ErrInvalidInput
	}
}

type UserPreferences struct {
	UserID             uuid.UUID
	FavoriteCategories []string
	FavoriteTags       []string
	WatchedVideos      []uuid.UUID
	LikedVideos        []uuid.UUID
	DislikedVideos     []uuid.UUID
	SubscribedCreators []uuid.UUID
}

type VideoSimilarity struct {
	VideoID        uuid.UUID
	SimilarVideoID uuid.UUID
	Score          float64
}
