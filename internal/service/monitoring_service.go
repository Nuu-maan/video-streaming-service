package service

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
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
	// INFO is called without section arguments. Passing multiple sections
	// ("INFO stats memory clients") is only supported from Redis 7.0 onward and
	// is a plain syntax error on anything older, so the previous call failed
	// outright against Redis 5 and 6. The argument-less form returns the default
	// sections — which include stats, memory, and clients — on every version.
	raw, err := s.redis.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("querying Redis INFO: %w", err)
	}
	info := parseRedisInfo(raw)

	dbSize, err := s.redis.DBSize(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("querying Redis DBSIZE: %w", err)
	}

	hits := info.int64("keyspace_hits")
	misses := info.int64("keyspace_misses")

	var hitRate float64
	if lookups := hits + misses; lookups > 0 {
		hitRate = float64(hits) / float64(lookups)
	}

	return &domain.RedisMetrics{
		MemoryUsed:       info.int64("used_memory"),
		MemoryPeak:       info.int64("used_memory_peak"),
		TotalKeys:        dbSize,
		Hits:             hits,
		Misses:           misses,
		HitRate:          hitRate,
		ConnectedClients: int(info.int64("connected_clients")),
		Timestamp:        time.Now(),
	}, nil
}

// redisInfo is the parsed form of the Redis INFO reply.
type redisInfo map[string]string

// int64 returns the named field as an int64, or 0 when the field is absent or
// unparseable. INFO fields are advisory metrics; a missing one is not an error.
func (i redisInfo) int64(field string) int64 {
	value, err := strconv.ParseInt(i[field], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// parseRedisInfo parses the INFO reply, which is CRLF-delimited "key:value"
// lines interleaved with "# Section" headers and blank lines.
func parseRedisInfo(raw string) redisInfo {
	info := make(redisInfo)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		info[key] = value
	}
	return info
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
