// Package app wires the API's dependencies and owns its lifecycle.
//
// Construction is separated from serving so the whole graph can be built in a
// test and exercised through its HTTP handler without binding a port.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Nuu-maan/video-streaming-service/internal/cache"
	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/handler"
	"github.com/Nuu-maan/video-streaming-service/internal/middleware"
	"github.com/Nuu-maan/video-streaming-service/internal/queue"
	"github.com/Nuu-maan/video-streaming-service/internal/repository/postgres"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/pkg/jwt"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// App owns every long-lived dependency of the API process.
type App struct {
	cfg *config.Config
	log *logger.Logger

	// startedAt backs the uptime reported by /health. The old /metrics stub
	// computed uptime as time.Since(time.Now()), which is always zero.
	startedAt time.Time

	db          *pgxpool.Pool
	redis       *redis.Client
	cache       *cache.CacheService
	queueClient *queue.QueueClient
	inspector   *asynq.Inspector

	authenticator *middleware.Authenticator
	rateLimiter   *middleware.RateLimiter

	authHandler       *handler.AuthHandler
	videoHandler      *handler.VideoHandler
	streamingHandler  *handler.StreamingHandler
	adminHandler      *handler.AdminHandler
	pageHandler       *handler.PageHandler
	analyticsHandler  *handler.AnalyticsHandler
	moderationHandler *handler.ModerationHandler
	monitoringHandler *handler.MonitoringHandler
}

// New builds the dependency graph. It returns a cleanly-closed App on error, so
// a failure partway through does not leak a database pool or Redis connection.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log := logger.New(cfg.Server.Environment, cfg.LogLevel)

	app := &App{cfg: cfg, log: log, startedAt: time.Now()}

	db, err := openDatabase(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	app.db = db

	redisClient, err := openRedis(ctx, cfg.Redis)
	if err != nil {
		app.Close()
		return nil, err
	}
	app.redis = redisClient

	// One inspector, shared. The admin handler used to construct its own from a
	// raw address string, so the process held two.
	app.inspector = asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	app.queueClient = queue.NewQueueClient(cfg.Redis.Address(), log)

	// Two-tier cache (in-process L1 in front of Redis). Playlists are small,
	// immutable once a video is ready, and requested once per viewer per few
	// seconds, so they benefit from the local tier.
	app.cache = cache.NewCacheService(redisClient, localCacheEntries)

	videoRepo := postgres.NewPostgresVideoRepository(db)
	userRepo := postgres.NewUserRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	auditRepo := postgres.NewAuditLogRepository(db)

	tokens := jwt.NewTokenService(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, cfg.Auth.JWTIssuer)
	app.authenticator = middleware.NewAuthenticator(tokens)
	app.rateLimiter = middleware.NewRateLimiter(redisClient)

	ffmpeg := service.NewFFmpegService(log)
	authService := service.NewAuthService(userRepo, tokens, cfg.Auth, log)
	uploadService := service.NewUploadService(videoRepo, ffmpeg, &cfg.Storage, log)
	auditService := service.NewAuditService(auditRepo)
	analyticsService := service.NewAnalyticsService(analyticsRepo, redisClient)
	moderationService := service.NewModerationService(reportRepo, videoRepo, userRepo, auditService)
	// Reuses the single inspector above rather than dialing Redis again.
	monitoringService := service.NewMonitoringService(db, redisClient, app.inspector)

	app.authHandler = handler.NewAuthHandler(authService, userRepo, log)
	app.videoHandler = handler.NewVideoHandler(uploadService, videoRepo, app.queueClient, log, cfg)
	app.streamingHandler = handler.NewStreamingHandler(videoRepo, app.cache, cfg, log)
	app.adminHandler = handler.NewAdminHandler(videoRepo, app.queueClient, cfg.Redis.Address(), log)
	app.pageHandler = handler.NewPageHandler(videoRepo, log)
	app.analyticsHandler = handler.NewAnalyticsHandler(analyticsService, log)
	app.moderationHandler = handler.NewModerationHandler(moderationService, log)
	app.monitoringHandler = handler.NewMonitoringHandler(monitoringService, log)

	return app, nil
}

// Run serves HTTP until ctx is cancelled, then shuts down gracefully within the
// configured timeout.
func (a *App) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              a.cfg.Server.Address(),
		Handler:           a.Handler(),
		ReadTimeout:       a.cfg.Server.ReadTimeout,
		WriteTimeout:      a.cfg.Server.WriteTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		a.log.Info(ctx, "HTTP server listening", map[string]interface{}{
			"address":     a.cfg.Server.Address(),
			"environment": a.cfg.Server.Environment,
		})
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		a.log.Info(context.WithoutCancel(ctx), "shutdown signal received", nil)
	}

	// ctx is already cancelled, so the shutdown deadline must hang off a live
	// context or Shutdown would return immediately without draining anything.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	a.log.Info(context.WithoutCancel(ctx), "server stopped cleanly", nil)
	return nil
}

// localCacheEntries bounds the in-process L1 cache.
const localCacheEntries = 10_000

// Close releases every dependency the App holds. It is safe to call on a
// partially-constructed App.
func (a *App) Close() {
	if a.cache != nil {
		// Stops the L1 eviction goroutine, which otherwise outlives the App.
		a.cache.Close()
	}
	if a.queueClient != nil {
		a.queueClient.Close()
	}
	if a.inspector != nil {
		a.inspector.Close()
	}
	if a.redis != nil {
		a.redis.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func openDatabase(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing database DSN: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	return pool, nil
}

func openRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("connecting to Redis: %w", err)
	}
	return client, nil
}

// ginMode maps the environment onto gin's mode.
func ginMode(environment string) string {
	if environment == config.EnvProduction {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}
