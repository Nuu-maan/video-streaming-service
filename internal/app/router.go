package app

import (
	"context"
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

	// gin trusts every proxy by default, which lets any client mint its own
	// ClientIP via X-Forwarded-For — the value the rate limiter and the view
	// dedupe key on. Trust only the proxies configuration names; none by
	// default, so ClientIP falls back to the unforgeable RemoteAddr. Entries
	// were syntax-checked by config.Validate, so an error here means gin and
	// config disagree about syntax — trusting nothing is the safe recovery.
	if err := router.SetTrustedProxies(a.cfg.Server.TrustedProxies); err != nil {
		a.log.Error(context.Background(), "invalid trusted proxy list; trusting none", err, nil)
		_ = router.SetTrustedProxies(nil)
	}

	router.Use(
		middleware.Recovery(a.log),
		middleware.RequestID(),
		middleware.Logger(a.log),
		middleware.CORS(a.cfg.CORS),
		metrics.MetricsMiddleware(),
	)

	a.registerOpsRoutes(router)
	a.registerPageRoutes(router)

	// The API surface is defined once and mounted twice: /api/v1 is the
	// canonical versioned prefix for external frontends, and bare /api is the
	// pre-versioning alias kept so nothing consuming today's paths breaks.
	a.registerAPIRoutes(router.Group("/api"))
	a.registerAPIRoutes(router.Group("/api/v1"))

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

	// API reference for external frontends: a self-contained HTML page (all
	// CSS/JS inline, no CDN assets, so it renders behind a strict CSP or with
	// no egress) plus the raw OpenAPI document it is generated from.
	router.GET("/docs", func(c *gin.Context) {
		c.File("./web/static/docs.html")
	})
	router.GET("/openapi.yaml", func(c *gin.Context) {
		// Set explicitly: Go's mime table has no entry for .yaml, and
		// http.ServeFile keeps a Content-Type that is already present.
		c.Header("Content-Type", "application/yaml; charset=utf-8")
		c.File("./docs/openapi.yaml")
	})

	// There is deliberately no static route over the uploads directory.
	//
	// One used to sit here, and it served the whole tree: /uploads/raw/<file>
	// handed out the untranscoded original, and /uploads/transcoded/<id>/... the
	// HLS output — neither of which consults the video's visibility. Every access
	// check in the streaming handler could be skipped simply by not going through
	// it, so a "private" video's playlist, thumbnail and original file were all
	// downloadable by anyone. It also could not work once the bytes moved to
	// object storage, which no file server can read.
	//
	// Media is served exclusively through /api/v1/videos/:id/... , which resolves
	// the video, checks who is asking, and reads through the storage backend.
}

func (a *App) registerPageRoutes(router *gin.Engine) {
	router.GET("/", a.pageHandler.UploadPage)
	router.GET("/videos", a.pageHandler.VideoListPage)
	router.GET("/videos/:id", a.pageHandler.VideoPlayerPage)
}

func (a *App) registerAPIRoutes(api *gin.RouterGroup) {
	auth := a.authenticator

	// Authentication and account recovery. Rate-limited harder than the rest
	// of the API: these are the endpoints worth brute-forcing, and the two
	// that send mail are amplification vectors on top.
	authRoutes := api.Group("/auth")
	authRoutes.Use(a.rateLimit("auth"))
	{
		authRoutes.POST("/register", a.authHandler.Register)
		authRoutes.POST("/login", a.authHandler.Login)
		authRoutes.POST("/refresh", a.authHandler.Refresh)
		authRoutes.GET("/me", auth.RequireAuth(), a.authHandler.Me)
		authRoutes.POST("/logout", auth.RequireAuth(), a.authHandler.Logout)
		authRoutes.POST("/logout-all", auth.RequireAuth(), a.authHandler.LogoutAll)

		authRoutes.POST("/verify-email/send", auth.RequireAuth(), a.accountHandler.SendVerificationEmail)
		authRoutes.POST("/verify-email", a.accountHandler.VerifyEmail)
		authRoutes.POST("/forgot-password", a.accountHandler.ForgotPassword)
		authRoutes.POST("/reset-password", a.accountHandler.ResetPassword)
	}

	videos := api.Group("/videos")
	videos.Use(a.rateLimit("user_api"))
	{
		// Listing and reading are public, but OptionalAuth lets an
		// authenticated caller additionally see their own private videos.
		videos.GET("", auth.OptionalAuth(), a.videoHandler.ListVideos)
		videos.GET("/trending", a.searchHandler.Trending)
		videos.GET("/:id", auth.OptionalAuth(), a.videoHandler.GetVideo)
		videos.GET("/:id/status", auth.OptionalAuth(), a.videoHandler.GetVideoStatus)
		videos.GET("/:id/related", a.searchHandler.Related)

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

		// A view may be anonymous — the handler then requires a session_id in
		// the body — but resume progress only means something for an account.
		videos.POST("/:id/view", auth.OptionalAuth(), a.viewHandler.RecordView)
		videos.POST("/:id/progress", auth.RequireAuth(), a.viewHandler.SaveProgress)

		videos.PUT("/:id/like", auth.RequireAuth(), a.socialHandler.SetLike)
		videos.DELETE("/:id/like", auth.RequireAuth(), a.socialHandler.RemoveLike)
		videos.GET("/:id/like", auth.RequireAuth(), a.socialHandler.GetLike)

		videos.GET("/:id/comments", a.socialHandler.ListComments)
		videos.POST("/:id/comments", auth.RequireAuth(), a.socialHandler.CreateComment)

		videos.PUT("/:id/watch-later", auth.RequireAuth(), a.socialHandler.AddWatchLater)
		videos.DELETE("/:id/watch-later", auth.RequireAuth(), a.socialHandler.RemoveWatchLater)
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

		// A thumbnail is a frame of the video, so it is exactly as private as the
		// video and is served under the same visibility check.
		streaming.GET("/thumbnail", a.streamingHandler.ServeThumbnail)
	}

	// Comment edits address the comment, not the video, so they carry their
	// own prefix. Delete authorisation (author, video owner, or moderator) is
	// resolved inside the handler — no route-level permission applies.
	comments := api.Group("/comments")
	comments.Use(a.rateLimit("user_api"))
	{
		comments.GET("/:id/replies", a.socialHandler.ListReplies)
		comments.PATCH("/:id", auth.RequireAuth(), a.socialHandler.UpdateComment)
		comments.DELETE("/:id", auth.RequireAuth(), a.socialHandler.DeleteComment)
	}

	users := api.Group("/users")
	users.Use(a.rateLimit("user_api"))
	{
		users.POST("/:id/subscribe", auth.RequireAuth(), a.socialHandler.Subscribe)
		users.DELETE("/:id/subscribe", auth.RequireAuth(), a.socialHandler.Unsubscribe)
		users.GET("/:id/subscribers", a.socialHandler.ListSubscribers)
	}

	playlists := api.Group("/playlists")
	playlists.Use(a.rateLimit("user_api"))
	{
		playlists.POST("", auth.RequireAuth(), a.socialHandler.CreatePlaylist)
		// OptionalAuth rather than none: a private playlist reads as 404 to
		// everyone but its owner, so the handler must know who is asking.
		playlists.GET("/:id", auth.OptionalAuth(), a.socialHandler.GetPlaylist)
		playlists.PATCH("/:id", auth.RequireAuth(), a.socialHandler.UpdatePlaylist)
		playlists.DELETE("/:id", auth.RequireAuth(), a.socialHandler.DeletePlaylist)

		playlists.POST("/:id/videos", auth.RequireAuth(), a.socialHandler.AddPlaylistVideo)
		playlists.DELETE("/:id/videos/:videoId", auth.RequireAuth(), a.socialHandler.RemovePlaylistVideo)
		playlists.GET("/:id/videos", auth.OptionalAuth(), a.socialHandler.ListPlaylistVideos)
	}

	// Discovery is public and read-only.
	discovery := api.Group("")
	discovery.Use(a.rateLimit("user_api"))
	{
		discovery.GET("/search", a.searchHandler.Search)
		discovery.GET("/search/suggest", a.searchHandler.Suggest)
		discovery.GET("/categories", a.searchHandler.Categories)
	}

	// The caller's own corner of the API.
	me := api.Group("/me")
	me.Use(a.rateLimit("user_api"), auth.RequireAuth())
	{
		me.GET("/feed", a.searchHandler.Feed)

		me.GET("/subscriptions", a.socialHandler.ListMySubscriptions)
		me.GET("/playlists", a.socialHandler.ListMyPlaylists)
		me.GET("/watch-later", a.socialHandler.ListWatchLater)

		me.GET("/history", a.viewHandler.GetHistory)
		me.DELETE("/history", a.viewHandler.ClearHistory)
		me.DELETE("/history/:videoId", a.viewHandler.DeleteHistoryEntry)

		me.GET("/notifications", a.socialHandler.ListNotifications)
		me.GET("/notifications/unread-count", a.socialHandler.UnreadNotificationCount)
		// The static route is registered before its parameterised sibling;
		// gin resolves both, but the order keeps the intent unambiguous.
		me.POST("/notifications/read-all", a.socialHandler.MarkAllNotificationsRead)
		me.POST("/notifications/:id/read", a.socialHandler.MarkNotificationRead)

		// The stricter auth budget applies on top of the group's: the request
		// body carries the account password, which makes it worth guessing at.
		me.POST("/change-password", a.rateLimit("auth"), a.accountHandler.ChangePassword)
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
	adminUsers := admin.Group("/users")
	adminUsers.Use(auth.RequirePermission(domain.PermissionManageUsers))
	{
		adminUsers.POST("/:id/ban", a.moderationHandler.BanUser)
		adminUsers.POST("/:id/unban", a.moderationHandler.UnbanUser)
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
	// The logging variant: a Redis outage fails open here, and a silently
	// disabled limiter is the kind of failure nobody notices until an abuse
	// report does.
	return middleware.RateLimitWithLogger(a.rateLimiter, rule, a.log)
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
