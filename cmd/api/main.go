package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/orchids/video-streaming/internal/config"
	"github.com/orchids/video-streaming/internal/handler"
	"github.com/orchids/video-streaming/internal/queue"
	"github.com/orchids/video-streaming/internal/repository/postgres"
	"github.com/orchids/video-streaming/internal/service"
	"github.com/orchids/video-streaming/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Server.Environment, cfg.LogLevel)
	log.Info(context.Background(), "Starting video streaming service", map[string]interface{}{
		"environment": cfg.Server.Environment,
		"port":        cfg.Server.Port,
	})

	dbPool, err := initDatabase(cfg)
	if err != nil {
		log.Fatal(context.Background(), "Failed to initialize database", err, nil)
	}
	defer dbPool.Close()
	log.Info(context.Background(), "Database connection established", nil)

	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatal(context.Background(), "Failed to initialize Redis", err, nil)
	}
	defer redisClient.Close()
	log.Info(context.Background(), "Redis connection established", nil)

	videoRepo := postgres.NewPostgresVideoRepository(dbPool)
	ffmpegService := service.NewFFmpegService(log)
	uploadService := service.NewUploadService(videoRepo, ffmpegService, &cfg.Storage, log)
	queueClient := queue.NewQueueClient(cfg.Redis.Address(), log)
	defer queueClient.Close()
	
	uploadHandler := handler.NewUploadHandler(uploadService, videoRepo, queueClient, log, cfg)
	pageHandler := handler.NewPageHandler(videoRepo, log)
	adminHandler := handler.NewAdminHandler(videoRepo, queueClient, cfg.Redis.Address(), log)
	streamingHandler := handler.NewStreamingHandler(videoRepo, redisClient, cfg, log)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestIDMiddleware())
	router.Use(LoggerMiddleware(log))
	router.Use(CORSMiddleware())

	router.MaxMultipartMemory = 10 << 20

	router.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()
		
		dbHealthy := true
		if err := dbPool.Ping(ctx); err != nil {
			dbHealthy = false
		}

		redisHealthy := true
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisHealthy = false
		}

		status := "healthy"
		httpStatus := http.StatusOK
		if !dbHealthy || !redisHealthy {
			status = "unhealthy"
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, gin.H{
			"status": status,
			"checks": gin.H{
				"database": dbHealthy,
				"redis":    redisHealthy,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "video-streaming-api",
			"version": "1.0.0",
			"uptime":  time.Since(time.Now()).String(),
		})
	})

	router.Static("/static", "./web/static")
	router.Static("/uploads", "./web/uploads")

	router.GET("/", pageHandler.UploadPage)
	router.GET("/videos", pageHandler.VideoListPage)
	router.GET("/videos/:id", pageHandler.VideoPlayerPage)

	api := router.Group("/api")
	{
		api.POST("/videos/upload", uploadHandler.Upload)
		api.GET("/videos", uploadHandler.ListVideos)
		api.GET("/videos/:id", uploadHandler.GetVideo)
		api.GET("/videos/:id/status", uploadHandler.GetVideoStatus)
		api.DELETE("/videos/:id", uploadHandler.DeleteVideo)
		
		api.GET("/videos/:id/hls/master.m3u8", streamingHandler.ServeMasterPlaylist)
		api.GET("/videos/:id/hls/:quality/playlist.m3u8", streamingHandler.ServeQualityPlaylist)
		api.GET("/videos/:id/hls/:quality/:segment", streamingHandler.ServeSegment)
		api.GET("/videos/:id/stream/:quality", streamingHandler.ServeMP4Fallback)
	}

	admin := router.Group("/api/admin")
	{
		admin.POST("/videos/:id/retry", adminHandler.RetryVideo)
		admin.GET("/queue/stats", adminHandler.GetQueueStats)
		admin.GET("/workers", adminHandler.ListActiveWorkers)
		admin.DELETE("/videos/:id/cache", streamingHandler.ClearPlaylistCache)
	}

	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info(context.Background(), "HTTP server starting", map[string]interface{}{
			"address": cfg.Server.Address(),
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(context.Background(), "Failed to start server", err, nil)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(context.Background(), "Shutting down server...", nil)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(context.Background(), "Server forced to shutdown", err, nil)
	}

	log.Info(context.Background(), "Server exited gracefully", nil)
}

func initDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Address(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis: %w", err)
	}

	return client, nil
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)
		
		c.Next()
	}
}

func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info(c.Request.Context(), "HTTP request", map[string]interface{}{
			"method":      method,
			"path":        path,
			"status":      statusCode,
			"latency_ms":  latency.Milliseconds(),
			"client_ip":   clientIP,
			"user_agent":  c.Request.UserAgent(),
		})
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
