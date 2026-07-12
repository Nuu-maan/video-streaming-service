package app

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/metrics"
	"github.com/Nuu-maan/video-streaming-service/internal/middleware"
)

// Handler builds the HTTP handler. It is exported so tests can drive the full
// routing stack through httptest without binding a port.
func (a *App) Handler() http.Handler {
	gin.SetMode(ginMode(a.cfg.Server.Environment))

	router := gin.New()
	router.MaxMultipartMemory = maxMultipartMemory

	router.Use(
		middleware.Recovery(a.log),
		middleware.RequestID(),
		middleware.Logger(a.log),
		middleware.CORS(a.cfg.CORS),
		metrics.MetricsMiddleware(),
	)

	a.registerOpsRoutes(router)
	a.registerPageRoutes(router)
	a.registerAPIRoutes(router)

	return router
}

// maxMultipartMemory bounds how much of an upload gin buffers in memory before
// spilling to a temp file.
const maxMultipartMemory = 10 << 20 // 10 MiB

func (a *App) registerOpsRoutes(router *gin.Engine) {
	router.GET("/health", a.health)

	// The real Prometheus exposition handler. This route previously returned a
	// hand-rolled JSON blob — which prometheus.yml was scraping and could never
	// parse — whose "uptime" field was time.Since(time.Now()), always ~0s.
	router.GET("/metrics", metrics.MetricsHandler())

	router.Static("/static", "./web/static")
	router.Static("/uploads", "./web/uploads")
}

func (a *App) registerPageRoutes(router *gin.Engine) {
	router.GET("/", a.pageHandler.UploadPage)
	router.GET("/videos", a.pageHandler.VideoListPage)
	router.GET("/videos/:id", a.pageHandler.VideoPlayerPage)
}

func (a *App) registerAPIRoutes(router *gin.Engine) {
	auth := a.authenticator

	api := router.Group("/api")

	// Authentication. Rate-limited harder than the rest of the API: these are
	// the endpoints worth brute-forcing.
	authRoutes := api.Group("/auth")
	authRoutes.Use(a.rateLimit("auth"))
	{
		authRoutes.POST("/register", a.authHandler.Register)
		authRoutes.POST("/login", a.authHandler.Login)
		authRoutes.POST("/refresh", a.authHandler.Refresh)
		authRoutes.GET("/me", auth.RequireAuth(), a.authHandler.Me)
	}

	videos := api.Group("/videos")
	videos.Use(a.rateLimit("user_api"))
	{
		// Listing and reading are public, but OptionalAuth lets an
		// authenticated caller additionally see their own private videos.
		videos.GET("", auth.OptionalAuth(), a.videoHandler.ListVideos)
		videos.GET("/:id", auth.OptionalAuth(), a.videoHandler.GetVideo)
		videos.GET("/:id/status", auth.OptionalAuth(), a.videoHandler.GetVideoStatus)

		// Writes require a caller. Upload was previously anonymous, so an
		// uploaded video had no owner and nobody could be held to it.
		videos.POST("/upload",
			auth.RequireAuth(),
			auth.RequirePermission(domain.PermissionUploadVideo),
			a.rateLimit("upload"),
			a.videoHandler.Upload,
		)
		// Ownership is enforced inside the handler: an owner may delete their
		// own video, and PermissionDeleteAnyVideo covers everyone else's.
		videos.DELETE("/:id", auth.RequireAuth(), a.videoHandler.DeleteVideo)
	}

	// Streaming. Kept in its own group with a far higher rate limit: a single
	// viewer pulls one HLS segment every few seconds.
	streaming := api.Group("/videos/:id")
	streaming.Use(a.rateLimit("streaming"), auth.OptionalAuth())
	{
		streaming.GET("/hls/master.m3u8", a.streamingHandler.ServeMasterPlaylist)
		streaming.GET("/hls/:quality/playlist.m3u8", a.streamingHandler.ServeQualityPlaylist)
		streaming.GET("/hls/:quality/:segment", a.streamingHandler.ServeSegment)
		streaming.GET("/stream/:quality", a.streamingHandler.ServeMP4Fallback)
	}

	// Reporting content is a user action, not a moderator one, so it hangs off
	// the public API surface behind nothing more than authentication.
	reports := api.Group("/reports")
	reports.Use(a.rateLimit("user_api"), auth.RequireAuth())
	{
		reports.POST("", a.moderationHandler.CreateReport)
	}

	// Admin. Every one of these routes was completely unauthenticated: anyone
	// on the network could inspect the queue, enumerate workers, and flush
	// caches without so much as a header.
	//
	// Authentication is common to the whole surface, but the permission is not:
	// applying PermissionModerateContent to the group handed moderators the
	// analytics and monitoring endpoints (which they do not hold
	// PermissionViewAnalytics for) and locked admins out of nothing. Each
	// subgroup now carries the permission it actually needs.
	admin := api.Group("/admin")
	admin.Use(auth.RequireAuth())

	ops := admin.Group("")
	ops.Use(auth.RequirePermission(domain.PermissionModerateContent))
	{
		ops.POST("/videos/:id/retry", a.adminHandler.RetryVideo)
		ops.GET("/queue/stats", a.adminHandler.GetQueueStats)
		ops.GET("/workers", a.adminHandler.ListActiveWorkers)
		ops.DELETE("/videos/:id/cache", a.streamingHandler.ClearPlaylistCache)

		ops.GET("/reports/pending", a.moderationHandler.ListPendingReports)
		ops.POST("/reports/:id/review", a.moderationHandler.ReviewReport)
	}

	// Banning is an account-lifecycle action, so it sits with user management
	// rather than with content moderation.
	users := admin.Group("/users")
	users.Use(auth.RequirePermission(domain.PermissionManageUsers))
	{
		users.POST("/:id/ban", a.moderationHandler.BanUser)
		users.POST("/:id/unban", a.moderationHandler.UnbanUser)
	}

	analytics := admin.Group("/analytics")
	analytics.Use(auth.RequirePermission(domain.PermissionViewAnalytics))
	{
		analytics.GET("/dashboard", a.analyticsHandler.GetDashboard)
		analytics.GET("/realtime", a.analyticsHandler.GetRealtimeMetrics)
		analytics.GET("/top-videos", a.analyticsHandler.GetTopVideos)
		analytics.GET("/videos/:id", a.analyticsHandler.GetVideoAnalytics)
		analytics.GET("/videos/:id/views", a.analyticsHandler.GetViewsTimeSeries)
	}

	// Monitoring exposes host, pool, and queue internals, so it is admin-only.
	monitoring := admin.Group("/monitoring")
	monitoring.Use(auth.RequirePermission(domain.PermissionManageUsers))
	{
		monitoring.GET("/metrics", a.monitoringHandler.GetAllMetrics)
		monitoring.GET("/system", a.monitoringHandler.GetSystemMetrics)
		monitoring.GET("/queue", a.monitoringHandler.GetQueueMetrics)
		monitoring.GET("/database", a.monitoringHandler.GetDatabaseMetrics)
		monitoring.GET("/redis", a.monitoringHandler.GetRedisMetrics)
	}
}

// rateLimit returns the limiter middleware for a rule, or a no-op when rate
// limiting is disabled. Enforcement lives here rather than in nginx, which only
// ever proxies MinIO and so never sees an API request.
func (a *App) rateLimit(rule string) gin.HandlerFunc {
	if !a.cfg.RateLimit.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	return middleware.RateLimitMiddleware(a.rateLimiter, rule)
}

// health reports readiness. It returns 503 when a dependency is unreachable so
// an orchestrator can pull the instance out of rotation.
func (a *App) health(c *gin.Context) {
	ctx := c.Request.Context()

	checks := gin.H{
		"database": a.db.Ping(ctx) == nil,
		"redis":    a.redis.Ping(ctx).Err() == nil,
	}

	status := http.StatusOK
	state := "healthy"
	for _, healthy := range checks {
		if healthy != true {
			status = http.StatusServiceUnavailable
			state = "unhealthy"
			break
		}
	}

	c.JSON(status, gin.H{
		"status":    state,
		"checks":    checks,
		"uptime":    time.Since(a.startedAt).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
