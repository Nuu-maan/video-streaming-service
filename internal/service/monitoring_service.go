package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type MonitoringService struct {
	db        *pgxpool.Pool
	redis     *redis.Client
	inspector *asynq.Inspector
	startTime time.Time
}

func NewMonitoringService(db *pgxpool.Pool, redisClient *redis.Client, inspector *asynq.Inspector) *MonitoringService {
	return &MonitoringService{
		db:        db,
		redis:     redisClient,
		inspector: inspector,
		startTime: time.Now(),
	}
}

func (s *MonitoringService) GetSystemMetrics(ctx context.Context) (*domain.SystemMetrics, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	diskStats, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	return &domain.SystemMetrics{
		CPUPercent:    cpuPercent[0],
		MemoryTotal:   memStats.Total,
		MemoryUsed:    memStats.Used,
		MemoryPercent: memStats.UsedPercent,
		DiskTotal:     diskStats.Total,
		DiskUsed:      diskStats.Used,
		DiskPercent:   diskStats.UsedPercent,
		Goroutines:    runtime.NumGoroutine(),
		Uptime:        time.Since(s.startTime),
		Timestamp:     time.Now(),
	}, nil
}

func (s *MonitoringService) GetQueueMetrics(ctx context.Context) (*domain.QueueMetrics, error) {
	queues, err := s.inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue list: %w", err)
	}

	var totalPending, totalActive, totalFailed, totalRetry, totalArchived, totalProcessed int64

	for _, queue := range queues {
		info, err := s.inspector.GetQueueInfo(queue)
		if err != nil {
			continue
		}
		totalPending += int64(info.Pending)
		totalActive += int64(info.Active)
		totalFailed += int64(info.Scheduled)
		totalArchived += int64(info.Archived)
		totalProcessed += int64(info.Processed)
	}

	return &domain.QueueMetrics{
		PendingJobs:   totalPending,
		ActiveJobs:    totalActive,
		FailedJobs:    totalFailed,
		RetryQueue:    totalRetry,
		ArchivedJobs:  totalArchived,
		ProcessedLast: totalProcessed,
		Timestamp:     time.Now(),
	}, nil
}

func (s *MonitoringService) GetDatabaseMetrics(ctx context.Context) (*domain.DatabaseMetrics, error) {
	stats := s.db.Stat()

	var slowQueries int64
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM pg_stat_statements 
		WHERE mean_exec_time > 1000
	`).Scan(&slowQueries)
	if err != nil {
		slowQueries = 0
	}

	tableSizes := make(map[string]int64)
	rows, err := s.db.Query(ctx, `
		SELECT 
			tablename,
			pg_total_relation_size(schemaname||'.'||tablename) AS size_bytes
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY size_bytes DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName string
			var sizeBytes int64
			if err := rows.Scan(&tableName, &sizeBytes); err == nil {
				tableSizes[tableName] = sizeBytes
			}
		}
	}

	return &domain.DatabaseMetrics{
		ActiveConnections: int(stats.AcquiredConns()),
		IdleConnections:   int(stats.IdleConns()),
		MaxConnections:    int(stats.MaxConns()),
		SlowQueries:       slowQueries,
		TotalQueries:      0,
		TableSizes:        tableSizes,
		Timestamp:         time.Now(),
	}, nil
}

func (s *MonitoringService) GetRedisMetrics(ctx context.Context) (*domain.RedisMetrics, error) {
	info, err := s.redis.Info(ctx, "stats", "memory", "clients").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	dbSize, err := s.redis.DBSize(ctx).Result()
	if err != nil {
		dbSize = 0
	}

	memoryUsed, _ := s.redis.Do(ctx, "INFO", "memory").Result()
	memoryPeak := int64(0)

	return &domain.RedisMetrics{
		MemoryUsed:       0,
		MemoryPeak:       memoryPeak,
		TotalKeys:        dbSize,
		Hits:             0,
		Misses:           0,
		HitRate:          0.0,
		ConnectedClients: 0,
		Timestamp:        time.Now(),
	}, nil
}

func (s *MonitoringService) CheckHealth(ctx context.Context) error {
	if err := s.db.Ping(ctx); err != nil {
		return fmt.Errorf("database unhealthy: %w", err)
	}

	if err := s.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis unhealthy: %w", err)
	}

	return nil
}

func (s *MonitoringService) GetAllMetrics(ctx context.Context) (map[string]interface{}, error) {
	systemMetrics, err := s.GetSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}

	queueMetrics, err := s.GetQueueMetrics(ctx)
	if err != nil {
		return nil, err
	}

	dbMetrics, err := s.GetDatabaseMetrics(ctx)
	if err != nil {
		return nil, err
	}

	redisMetrics, err := s.GetRedisMetrics(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"system":   systemMetrics,
		"queue":    queueMetrics,
		"database": dbMetrics,
		"redis":    redisMetrics,
	}, nil
}
