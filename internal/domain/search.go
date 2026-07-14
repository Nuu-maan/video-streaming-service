package domain

import (
	"time"

	"github.com/google/uuid"
)

type DurationFilter string

const (
	DurationShort  DurationFilter = "short"  // < 4 minutes
	DurationMedium DurationFilter = "medium" // 4-20 minutes
	DurationLong   DurationFilter = "long"   // > 20 minutes
)

type DateFilter string

const (
	DateToday DateFilter = "today"
	DateWeek  DateFilter = "week"
	DateMonth DateFilter = "month"
	DateYear  DateFilter = "year"
)

type SearchFilters struct {
	Duration   *DurationFilter
	UploadDate *DateFilter
	Quality    []string
	Categories []string
	Tags       []string
	// MinDuration/MaxDuration are exact bounds in seconds. They coexist with the
	// Duration presets because the API exposes numeric bounds while the presets
	// remain for callers that want the coarse buckets.
	MinDuration  *int
	MaxDuration  *int
	MinViews     *int64
	MaxViews     *int64
	OnlyVerified bool
	Language     *string
}

type SearchQuery struct {
	Query   string
	Filters SearchFilters
	SortBy  string // "relevance", "newest", "views", "likes"
	Page    int
	Limit   int
}

type VideoSearchItem struct {
	VideoID       uuid.UUID `json:"video_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ThumbnailURL  string    `json:"thumbnail_url"`
	Duration      int32     `json:"duration"`
	Views         int64     `json:"views"`
	CreatedAt     time.Time `json:"created_at"`
	Username      string    `json:"username"`
	UserID        uuid.UUID `json:"user_id"`
	UserAvatarURL string    `json:"user_avatar_url"`
	UserVerified  bool      `json:"user_verified"`
	Relevance     float64   `json:"relevance"`
	Snippet       string    `json:"snippet"`
}

type SearchFacets struct {
	Categories  map[string]int64 `json:"categories"`
	Durations   map[string]int64 `json:"durations"`
	UploadDates map[string]int64 `json:"upload_dates"`
}

// CategoryCount is one row of the public category index: a category name and
// how many publicly listable videos carry it.
type CategoryCount struct {
	Category   string `json:"category"`
	VideoCount int64  `json:"video_count"`
}

type SearchResult struct {
	Videos     []*VideoSearchItem `json:"videos"`
	TotalCount int64              `json:"total_count"`
	Facets     *SearchFacets      `json:"facets"`
	Query      string             `json:"query"`
	Took       time.Duration      `json:"took"`
	Page       int                `json:"page"`
	TotalPages int                `json:"total_pages"`
}

func (sq *SearchQuery) Validate() error {
	if sq.Query == "" && len(sq.Filters.Categories) == 0 && len(sq.Filters.Tags) == 0 {
		return ErrInvalidInput
	}

	if sq.Page < 1 {
		sq.Page = 1
	}

	if sq.Limit < 1 || sq.Limit > 100 {
		sq.Limit = 20
	}

	if sq.SortBy == "" {
		sq.SortBy = "relevance"
	}

	validSortBy := map[string]bool{
		"relevance": true,
		"newest":    true,
		"views":     true,
		"likes":     true,
	}

	if !validSortBy[sq.SortBy] {
		return ErrInvalidInput
	}

	return nil
}

func (df DurationFilter) ToSeconds() (min, max int32) {
	switch df {
	case DurationShort:
		return 0, 240 // 4 minutes
	case DurationMedium:
		return 240, 1200 // 4-20 minutes
	case DurationLong:
		return 1200, 999999 // 20+ minutes
	default:
		return 0, 999999
	}
}

func (datef DateFilter) ToTime() time.Time {
	now := time.Now()
	switch datef {
	case DateToday:
		return now.AddDate(0, 0, -1)
	case DateWeek:
		return now.AddDate(0, 0, -7)
	case DateMonth:
		return now.AddDate(0, -1, 0)
	case DateYear:
		return now.AddDate(-1, 0, 0)
	default:
		return time.Time{}
	}
}
