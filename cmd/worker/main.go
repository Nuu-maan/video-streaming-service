package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/orchids/video-streaming/internal/config"
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
	log.Info(context.Background(), "Starting video processing worker", map[string]interface{}{
		"environment": cfg.Server.Environment,
		"concurrency": cfg.Worker.MaxConcurrentJobs,
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
	transcodingService := service.NewTranscodingService(videoRepo, ffmpegService, &cfg.Storage, log)

	videoProcessingHandler := queue.NewVideoProcessingHandler(transcodingService, log)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Address()},
		asynq.Config{
			Concurrency: cfg.Worker.MaxConcurrentJobs,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error(ctx, "task execution failed", map[string]interface{}{
					"task_type": task.Type(),
					"task_id":   task.ResultWriter().TaskID(),
					"error":     err.Error(),
					"payload":   string(task.Payload()),
				})
			}),
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				delays := []time.Duration{
					1 * time.Minute,
					5 * time.Minute,
					30 * time.Minute,
				}
				if n < len(delays) {
					return delays[n]
				}
				return delays[len(delays)-1]
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeVideoProcessing, videoProcessingHandler.ProcessTask)

	go func() {
		log.Info(context.Background(), "Worker server starting", map[string]interface{}{
			"concurrency": cfg.Worker.MaxConcurrentJobs,
		})
		if err := srv.Run(mux); err != nil {
			log.Fatal(context.Background(), "Worker server failed", err, nil)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(context.Background(), "Shutting down worker server...", nil)

	srv.Shutdown()

	log.Info(context.Background(), "Worker server exited gracefully", nil)
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
