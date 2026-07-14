package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

const (
	// suggestLimit caps autocomplete responses; a dropdown showing more than
	// ten entries is noise, and the trigram scan cost grows with the limit.
	suggestLimit = 10

	// relatedFallbackWindowHours is the trending window used to top up a short
	// related-videos rail. A week rather than a day: the fill exists to keep
	// the rail from looking empty, and a 24h window on a quiet instance is
	// itself often empty.
	relatedFallbackWindowHours = 7 * 24

	defaultDiscoveryLimit = 20
	maxDiscoveryLimit     = 100
)

// SearchRepository is the slice of the search store this service needs.
// Satisfied by *postgres.SearchRepository (asserted in that package).
type SearchRepository interface {
	Search(ctx context.Context, query *domain.SearchQuery) ([]*domain.VideoSearchItem, int64, error)
	Suggest(ctx context.Context, prefix string, limit int) ([]string, error)
	Trending(ctx context.Context, windowHours, limit int, exclude []uuid.UUID) ([]*domain.VideoSearchItem, error)
	Related(ctx context.Context, videoID uuid.UUID, limit int) ([]*domain.VideoSearchItem, error)
	Feed(ctx context.Context, subscriberID uuid.UUID, limit, offset int) ([]*domain.VideoSearchItem, int64, error)
	Categories(ctx context.Context) ([]*domain.CategoryCount, error)
}

type SearchService struct {
	repo SearchRepository
}

func NewSearchService(repo SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

// Search runs a validated full-text query and assembles the result envelope.
func (s *SearchService) Search(ctx context.Context, query *domain.SearchQuery) (*domain.SearchResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	start := time.Now()
	items, total, err := s.repo.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("searching videos: %w", err)
	}

	return &domain.SearchResult{
		Videos:     items,
		TotalCount: total,
		Query:      query.Query,
		Took:       time.Since(start),
		Page:       query.Page,
		TotalPages: int((total + int64(query.Limit) - 1) / int64(query.Limit)),
	}, nil
}

func (s *SearchService) Suggest(ctx context.Context, prefix string) ([]string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, domain.ErrInvalidInput
	}

	titles, err := s.repo.Suggest(ctx, prefix, suggestLimit)
	if err != nil {
		return nil, fmt.Errorf("suggesting titles: %w", err)
	}
	return titles, nil
}

func (s *SearchService) Trending(ctx context.Context, window domain.TrendingWindow, limit int) ([]*domain.VideoSearchItem, error) {
	if err := window.Validate(); err != nil {
		return nil, err
	}

	items, err := s.repo.Trending(ctx, window.Hours(), clampLimit(limit), nil)
	if err != nil {
		return nil, fmt.Errorf("listing trending videos: %w", err)
	}
	return items, nil
}

// Related returns content-similar videos for videoID. When metadata similarity
// cannot fill the requested count — a sparse catalogue, or a source with no
// tags and no category — the remainder is topped up from trending, excluding
// the source and anything already picked.
func (s *SearchService) Related(ctx context.Context, videoID uuid.UUID, limit int) ([]*domain.VideoSearchItem, error) {
	limit = clampLimit(limit)

	items, err := s.repo.Related(ctx, videoID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing related videos: %w", err)
	}
	if len(items) >= limit {
		return items, nil
	}

	exclude := make([]uuid.UUID, 0, len(items)+1)
	exclude = append(exclude, videoID)
	for _, item := range items {
		exclude = append(exclude, item.VideoID)
	}

	fill, err := s.repo.Trending(ctx, relatedFallbackWindowHours, limit-len(items), exclude)
	if err != nil {
		return nil, fmt.Errorf("topping up related videos from trending: %w", err)
	}
	return append(items, fill...), nil
}

func (s *SearchService) Feed(ctx context.Context, subscriberID uuid.UUID, limit, offset int) ([]*domain.VideoSearchItem, int64, error) {
	items, total, err := s.repo.Feed(ctx, subscriberID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing subscription feed: %w", err)
	}
	return items, total, nil
}

func (s *SearchService) Categories(ctx context.Context) ([]*domain.CategoryCount, error) {
	categories, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return categories, nil
}

func clampLimit(limit int) int {
	if limit < 1 || limit > maxDiscoveryLimit {
		return defaultDiscoveryLimit
	}
	return limit
}
