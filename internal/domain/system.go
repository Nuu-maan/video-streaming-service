package domain

import "time"

type SystemMetrics struct {
	CPUPercent    float64
	MemoryTotal   uint64
	MemoryUsed    uint64
	MemoryPercent float64
	DiskTotal     uint64
	DiskUsed      uint64
	DiskPercent   float64
	Goroutines    int
	Uptime        time.Duration
	Timestamp     time.Time
}

type QueueMetrics struct {
	PendingJobs   int64
	ActiveJobs    int64
	FailedJobs    int64
	RetryQueue    int64
	ArchivedJobs  int64
	ProcessedLast int64
	Timestamp     time.Time
}

type DatabaseMetrics struct {
	ActiveConnections int
	IdleConnections   int
	MaxConnections    int
	SlowQueries       int64
	TotalQueries      int64
	TableSizes        map[string]int64
	Timestamp         time.Time
}

type RedisMetrics struct {
	MemoryUsed      int64
	MemoryPeak      int64
	TotalKeys       int64
	Hits            int64
	Misses          int64
	HitRate         float64
	ConnectedClients int
	Timestamp       time.Time
}
