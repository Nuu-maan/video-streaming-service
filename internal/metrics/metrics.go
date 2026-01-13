package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	videosUploaded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "videos_uploaded_total",
			Help: "Total number of videos uploaded",
		},
	)

	videosProcessing = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "videos_processing_current",
			Help: "Current number of videos being processed",
		},
	)

	videosProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "videos_processed_total",
			Help: "Total number of videos processed",
		},
		[]string{"status", "quality"},
	)

	videoProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "video_processing_duration_seconds",
			Help:    "Video processing duration in seconds",
			Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
		},
		[]string{"quality"},
	)

	videoViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "video_views_total",
			Help: "Total number of video views",
		},
		[]string{"quality"},
	)

	videoStreamingBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "video_streaming_bytes_total",
			Help: "Total bytes streamed",
		},
		[]string{"quality"},
	)

	dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total database queries",
		},
		[]string{"operation", "table"},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5},
		},
		[]string{"operation", "table"},
	)

	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_type"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_type"},
	)

	cacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"operation"},
	)

	storageOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_operations_total",
			Help: "Total storage operations",
		},
		[]string{"operation", "bucket"},
	)

	storageBytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_bytes_transferred_total",
			Help: "Total bytes transferred to/from storage",
		},
		[]string{"direction", "bucket"},
	)

	queueJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_jobs_total",
			Help: "Total queue jobs",
		},
		[]string{"queue", "status"},
	)

	queueJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queue_job_duration_seconds",
			Help:    "Queue job duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
		[]string{"queue"},
	)

	queueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_size_current",
			Help: "Current queue size",
		},
		[]string{"queue"},
	)

	usersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of registered users",
		},
	)

	usersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_active_current",
			Help: "Number of currently active users",
		},
	)

	authAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total authentication attempts",
		},
		[]string{"type", "status"},
	)

	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
		[]string{"rule"},
	)
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		activeConnections.Inc()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		httpResponseSize.WithLabelValues(c.Request.Method, path).Observe(float64(c.Writer.Size()))

		if c.Request.ContentLength > 0 {
			httpRequestSize.WithLabelValues(c.Request.Method, path).Observe(float64(c.Request.ContentLength))
		}

		activeConnections.Dec()
	}
}

func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func RecordVideoUpload() {
	videosUploaded.Inc()
}

func RecordVideoProcessingStart() {
	videosProcessing.Inc()
}

func RecordVideoProcessingEnd(status, quality string, duration time.Duration) {
	videosProcessing.Dec()
	videosProcessed.WithLabelValues(status, quality).Inc()
	videoProcessingDuration.WithLabelValues(quality).Observe(duration.Seconds())
}

func RecordVideoView(quality string) {
	videoViews.WithLabelValues(quality).Inc()
}

func RecordVideoStreaming(quality string, bytes int64) {
	videoStreamingBytes.WithLabelValues(quality).Add(float64(bytes))
}

func RecordDBQuery(operation, table string, duration time.Duration) {
	dbQueriesTotal.WithLabelValues(operation, table).Inc()
	dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func SetDBConnections(active, idle int) {
	dbConnectionsActive.Set(float64(active))
	dbConnectionsIdle.Set(float64(idle))
}

func RecordCacheHit(cacheType string) {
	cacheHits.WithLabelValues(cacheType).Inc()
}

func RecordCacheMiss(cacheType string) {
	cacheMisses.WithLabelValues(cacheType).Inc()
}

func RecordCacheOperation(operation string, duration time.Duration) {
	cacheOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func RecordStorageOperation(operation, bucket string) {
	storageOperationsTotal.WithLabelValues(operation, bucket).Inc()
}

func RecordStorageTransfer(direction, bucket string, bytes int64) {
	storageBytesTransferred.WithLabelValues(direction, bucket).Add(float64(bytes))
}

func RecordQueueJob(queue, status string) {
	queueJobsTotal.WithLabelValues(queue, status).Inc()
}

func RecordQueueJobDuration(queue string, duration time.Duration) {
	queueJobDuration.WithLabelValues(queue).Observe(duration.Seconds())
}

func SetQueueSize(queue string, size int) {
	queueSize.WithLabelValues(queue).Set(float64(size))
}

func SetUsersTotal(count int) {
	usersTotal.Set(float64(count))
}

func SetUsersActive(count int) {
	usersActive.Set(float64(count))
}

func RecordAuthAttempt(authType, status string) {
	authAttemptsTotal.WithLabelValues(authType, status).Inc()
}

func RecordRateLimitHit(rule string) {
	rateLimitHits.WithLabelValues(rule).Inc()
}
