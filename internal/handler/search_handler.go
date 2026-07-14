package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

type SearchHandler struct {
	search *service.SearchService
	log    *logger.Logger
}

func NewSearchHandler(search *service.SearchService, log *logger.Logger) *SearchHandler {
	return &SearchHandler{search: search, log: log}
}

// Search runs a full-text video search. Public: it only ever surfaces public,
// ready videos, so there is nothing to gate on authentication.
func (h *SearchHandler) Search(c *gin.Context) {
	ctx := c.Request.Context()

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.ValidationError(c, "q is required")
		return
	}

	page := parsePage(c)
	query := &domain.SearchQuery{
		Query:  q,
		SortBy: strings.TrimSpace(c.DefaultQuery("sort", "relevance")),
		Page:   page.Offset/page.Limit + 1,
		Limit:  page.Limit,
	}

	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query.Filters.Categories = []string{category}
	}
	if language := strings.TrimSpace(c.Query("language")); language != "" {
		query.Filters.Language = &language
	}
	query.Filters.Tags = splitTags(c.Query("tags"))

	var bad bool
	if query.Filters.MinDuration, bad = optionalNonNegativeInt(c, "min_duration"); bad {
		return
	}
	if query.Filters.MaxDuration, bad = optionalNonNegativeInt(c, "max_duration"); bad {
		return
	}

	result, err := h.search.Search(ctx, query)
	if err != nil {
		// q is known non-empty, so a validation failure can only be the sort.
		if errors.Is(err, domain.ErrInvalidInput) {
			response.ValidationError(c, "sort must be one of: relevance, newest, views, likes")
			return
		}
		h.log.Error(ctx, "search failed", err, map[string]interface{}{"query": q})
		response.InternalError(c, "Search failed")
		return
	}

	// Search is a paginated list, so it answers in the same envelope as every
	// other paginated list. It used to return its own shape — the items nested
	// under data.videos, the total as data.total_count — which forced a client to
	// special-case this one endpoint for no reason a caller could see.
	response.SuccessWithList(c, result.Videos, paginationMeta(int(result.TotalCount), page))
}

// Suggest returns up to ten title suggestions for autocomplete, as bare
// strings: this endpoint is hit on every keystroke, so the payload stays
// minimal.
func (h *SearchHandler) Suggest(c *gin.Context) {
	ctx := c.Request.Context()

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.ValidationError(c, "q is required")
		return
	}

	suggestions, err := h.search.Suggest(ctx, q)
	if err != nil {
		h.log.Error(ctx, "suggest failed", err, map[string]interface{}{"query": q})
		response.InternalError(c, "Failed to load suggestions")
		return
	}

	response.Success(c, http.StatusOK, suggestions)
}

// Trending lists the most engaged-with public videos inside a time window.
func (h *SearchHandler) Trending(c *gin.Context) {
	ctx := c.Request.Context()

	window := domain.TrendingWindow(strings.TrimSpace(c.DefaultQuery("window", string(domain.TrendingWindow24h))))

	items, err := h.search.Trending(ctx, window, parsePage(c).Limit)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			response.ValidationError(c, "window must be one of: 24h, 7d, 30d")
			return
		}
		h.log.Error(ctx, "trending failed", err, map[string]interface{}{"window": window})
		response.InternalError(c, "Failed to load trending videos")
		return
	}

	response.Success(c, http.StatusOK, items)
}

// Related lists videos similar to the given one by shared tags and category,
// topped up from trending when metadata similarity runs short.
func (h *SearchHandler) Related(c *gin.Context) {
	ctx := c.Request.Context()

	videoID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	items, err := h.search.Related(ctx, videoID, parsePage(c).Limit)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "related videos failed", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to load related videos")
		return
	}

	response.Success(c, http.StatusOK, items)
}

// Feed lists videos from creators the caller subscribes to, newest first.
func (h *SearchHandler) Feed(c *gin.Context) {
	ctx := c.Request.Context()

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	page := parsePage(c)
	items, total, err := h.search.Feed(ctx, principal.UserID, page.Limit, page.Offset)
	if err != nil {
		h.log.Error(ctx, "feed failed", err, nil)
		response.InternalError(c, "Failed to load feed")
		return
	}

	response.SuccessWithList(c, items, paginationMeta(int(total), page))
}

// Categories lists the distinct categories in use with their video counts.
func (h *SearchHandler) Categories(c *gin.Context) {
	ctx := c.Request.Context()

	categories, err := h.search.Categories(ctx)
	if err != nil {
		h.log.Error(ctx, "categories failed", err, nil)
		response.InternalError(c, "Failed to load categories")
		return
	}

	response.Success(c, http.StatusOK, categories)
}

// splitTags parses a comma-separated tag list, dropping empties so trailing
// commas and doubled separators don't turn into impossible-to-match "" tags.
func splitTags(raw string) []string {
	var tags []string
	for _, tag := range strings.Split(raw, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// optionalNonNegativeInt parses an optional numeric query parameter. It
// returns nil for an absent value, and reports true when the value was present
// but invalid — in which case the error response has already been written.
func optionalNonNegativeInt(c *gin.Context, name string) (*int, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, false
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		response.ValidationError(c, name+" must be a non-negative integer")
		return nil, true
	}
	return &value, false
}
