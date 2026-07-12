package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
)

type MonitoringHandler struct {
	monitoring *service.MonitoringService
	log        *logger.Logger
}

func NewMonitoringHandler(monitoring *service.MonitoringService, log *logger.Logger) *MonitoringHandler {
	return &MonitoringHandler{monitoring: monitoring, log: log}
}

// GetAllMetrics returns system, queue, database, and Redis metrics in one
// payload. Every failure here is a failure of the process or one of its
// dependencies, never of the request, so there is nothing to map onto a 4xx.
func (h *MonitoringHandler) GetAllMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.monitoring.GetAllMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to collect metrics", err, nil)
		response.InternalError(c, "Failed to collect metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetSystemMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.monitoring.GetSystemMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to collect system metrics", err, nil)
		response.InternalError(c, "Failed to collect system metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetQueueMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.monitoring.GetQueueMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to collect queue metrics", err, nil)
		response.InternalError(c, "Failed to collect queue metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetDatabaseMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.monitoring.GetDatabaseMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to collect database metrics", err, nil)
		response.InternalError(c, "Failed to collect database metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetRedisMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	metrics, err := h.monitoring.GetRedisMetrics(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to collect Redis metrics", err, nil)
		response.InternalError(c, "Failed to collect Redis metrics")
		return
	}

	response.Success(c, http.StatusOK, metrics)
}
