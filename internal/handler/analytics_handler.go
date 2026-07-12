package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

const (
	defaultTopVideosLimit = 10
	maxTopVideosLimit     = 50
)

type AnalyticsHandler struct {
	analytics *service.AnalyticsService
	log       *logger.Logger
}

func NewAnalyticsHandler(analytics *service.AnalyticsService, log *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics, log: log}
}

// GetDashboard returns the platform-wide overview shown on the admin dashboard.
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.analytics.GetDashboardOverview(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to load dashboard analytics", err, nil)
		response.InternalError(c, "Failed to retrieve dashboard analytics")
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// GetVideoAnalytics returns the engagement breakdown for one video.
func (h *AnalyticsHandler) GetVideoAnalytics(c *gin.Context) {
	ctx := c.Request.Context()

	videoID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	principal, ok := appctx.PrincipalFrom(ctx)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}

	analytics, err := h.analytics.GetVideoAnalytics(ctx, videoID, principal.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to load video analytics", err, map[string]interface{}{"video_id": videoID})
		response.InternalError(c, "Failed to retrieve video analytics")
		return
	}

	response.Success(c, http.StatusOK, analytics)
}

// GetTopVideos returns the most-viewed videos of the past week.
func (h *AnalyticsHandler) GetTopVideos(c *gin.Context) {
	ctx := c.Request.Context()

	limit, ok := parseLimit(c, defaultTopVideosLimit, maxTopVideosLimit)
	if !ok {
		return
	}

	videos, err := h.analytics.GetTopVideosThisWeek(ctx, limit)
	if err != nil {
		h.log.Error(ctx, "failed to load top videos", err, map[string]interface{}{"limit": limit})
		response.InternalError(c, "Failed to retrieve top videos")
		return
	}

	response.Success(c, http.StatusOK, videos)
}

// GetRealtimeMetrics returns the live counters. It is deliberately uncached in
// the service, so it is the one analytics endpoint that always hits the database.
func (h *AnalyticsHandler) GetRealtimeMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.analytics.GetRealtimeMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to load realtime metrics", err, nil)
		response.InternalError(c, "Failed to retrieve realtime metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}

// GetViewsTimeSeries returns a view count series for a video, bucketed by the
// requested interval.
func (h *AnalyticsHandler) GetViewsTimeSeries(c *gin.Context) {
	ctx := c.Request.Context()

	videoID, err := validator.ValidateUUID(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	interval := c.DefaultQuery("interval", "hour")
	if !isValidInterval(interval) {
		response.ValidationError(c, "interval must be one of: hour, day, week, month")
		return
	}

	series, err := h.analytics.GetViewsTimeSeries(ctx, videoID, interval)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to load views time series", err, map[string]interface{}{
			"video_id": videoID,
			"interval": interval,
		})
		response.InternalError(c, "Failed to retrieve views time series")
		return
	}

	response.Success(c, http.StatusOK, series)
}

// isValidInterval guards the interval before it reaches the query, which
// bucketing depends on. Anything else would silently fall back to hourly.
func isValidInterval(interval string) bool {
	switch interval {
	case "hour", "day", "week", "month":
		return true
	}
	return false
}

// parseLimit reads a bounded "limit" query parameter. Unlike parsePage it
// rejects a malformed value rather than defaulting, so a caller who asked for
// something impossible is told so.
func parseLimit(c *gin.Context, fallback, max int) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return fallback, true
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > max {
		response.ValidationError(c, "limit must be a number between 1 and "+strconv.Itoa(max))
		return 0, false
	}

	return limit, true
}
