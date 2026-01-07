package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Host            string
	Port            string
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

type StorageConfig struct {
	UploadPath       string
	MaxFileSize      int64
	AllowedFormats   []string
	ThumbnailPath    string
	TranscodedPath   string
}

type WorkerConfig struct {
	MaxConcurrentJobs int
	JobTimeout        time.Duration
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Storage  StorageConfig
	Worker   WorkerConfig
	LogLevel string
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", "8080"),
			Environment:     getEnv("ENVIRONMENT", "development"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "video_streaming"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getIntEnv("REDIS_DB", 0),
			PoolSize:     getIntEnv("REDIS_POOL_SIZE", 10),
			MinIdleConns: getIntEnv("REDIS_MIN_IDLE_CONNS", 2),
		},
		Storage: StorageConfig{
			UploadPath:     getEnv("STORAGE_UPLOAD_PATH", "./web/uploads"),
			MaxFileSize:    getInt64Env("STORAGE_MAX_FILE_SIZE", 2*1024*1024*1024),
			AllowedFormats: []string{"video/mp4", "video/mpeg", "video/quicktime", "video/webm"},
			ThumbnailPath:  getEnv("STORAGE_THUMBNAIL_PATH", "./web/uploads/thumbnails"),
			TranscodedPath: getEnv("STORAGE_TRANSCODED_PATH", "./web/uploads/transcoded"),
		},
		Worker: WorkerConfig{
			MaxConcurrentJobs: getIntEnv("WORKER_MAX_CONCURRENT_JOBS", 4),
			JobTimeout:        getDurationEnv("WORKER_JOB_TIMEOUT", 30*time.Minute),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.Host == "" || c.Database.DBName == "" {
		return fmt.Errorf("database configuration is incomplete")
	}
	if c.Storage.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
