package domain

import "time"

// Operational metrics returned by the monitoring endpoints.
//
// The json tags are explicit and snake_case so these serialize consistently with
// the rest of the API. Without them Go emits its field names verbatim, and the
// monitoring endpoints alone answered in PascalCase.

type SystemMetrics struct {
	CPUPercent    float64       `json:"cpu_percent"`
	MemoryTotal   uint64        `json:"memory_total"`
	MemoryUsed    uint64        `json:"memory_used"`
	MemoryPercent float64       `json:"memory_percent"`
	DiskTotal     uint64        `json:"disk_total"`
	DiskUsed      uint64        `json:"disk_used"`
	DiskPercent   float64       `json:"disk_percent"`
	Goroutines    int           `json:"goroutines"`
	Uptime        time.Duration `json:"uptime"`
	Timestamp     time.Time     `json:"timestamp"`
}

type QueueMetrics struct {
	PendingJobs   int64     `json:"pending_jobs"`
	ActiveJobs    int64     `json:"active_jobs"`
	FailedJobs    int64     `json:"failed_jobs"`
	RetryQueue    int64     `json:"retry_queue"`
	ArchivedJobs  int64     `json:"archived_jobs"`
	ProcessedLast int64     `json:"processed_last"`
	Timestamp     time.Time `json:"timestamp"`
}

type DatabaseMetrics struct {
	ActiveConnections int              `json:"active_connections"`
	IdleConnections   int              `json:"idle_connections"`
	MaxConnections    int              `json:"max_connections"`
	SlowQueries       int64            `json:"slow_queries"`
	TotalQueries      int64            `json:"total_queries"`
	TableSizes        map[string]int64 `json:"table_sizes"`
	Timestamp         time.Time        `json:"timestamp"`
}

type RedisMetrics struct {
	MemoryUsed       int64     `json:"memory_used"`
	MemoryPeak       int64     `json:"memory_peak"`
	TotalKeys        int64     `json:"total_keys"`
	Hits             int64     `json:"hits"`
	Misses           int64     `json:"misses"`
	HitRate          float64   `json:"hit_rate"`
	ConnectedClients int       `json:"connected_clients"`
	Timestamp        time.Time `json:"timestamp"`
}
