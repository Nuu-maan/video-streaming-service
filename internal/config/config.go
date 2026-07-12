// Package config loads and validates application configuration from the
// environment. Every setting has a default suitable for local development;
// production is held to a stricter standard by Validate.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// EnvProduction is the ENVIRONMENT value that switches on production-grade
// validation: secrets must be set explicitly and TLS must not be disabled.
const EnvProduction = "production"

// insecureDefaultJWTSecret is the development-only signing key. Validate
// rejects it in production so a deploy can never silently sign tokens with a
// value that is public in this repository.
const insecureDefaultJWTSecret = "dev-only-insecure-jwt-secret-change-me"

type ServerConfig struct {
	Host            string
	Port            string
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// IsProduction reports whether the service is running in production.
func (c ServerConfig) IsProduction() bool {
	return c.Environment == EnvProduction
}

// Address returns the host:port the HTTP server binds to.
func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
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

// DSN returns a PostgreSQL connection URL. The user and password are
// percent-encoded: the previous keyword/value form broke outright on passwords
// containing spaces or quotes.
func (c DatabaseConfig) DSN() string {
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%s", c.Host, c.Port),
		Path:     c.DBName,
		RawQuery: url.Values{"sslmode": {c.SSLMode}}.Encode(),
	}
	return dsn.String()
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// Address returns the host:port of the Redis server.
func (c RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

type StorageConfig struct {
	UploadPath     string
	MaxFileSize    int64
	AllowedFormats []string
	ThumbnailPath  string
	TranscodedPath string
}

// AuthConfig governs token issuance and password handling. There was no auth
// configuration at all before; JWT signing had no key to use.
type AuthConfig struct {
	JWTSecret       string
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// CORSConfig lists the origins permitted to make credentialed requests.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         time.Duration
}

// AllowsOrigin reports whether origin may make a credentialed cross-origin
// request. A configured "*" allows any origin, but only because Validate
// forbids that combination in production: the browser rejects wildcard origin
// together with credentials, and honouring it server-side would be a CSRF hole.
func (c CORSConfig) AllowsOrigin(origin string) bool {
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// AllowsAnyOrigin reports whether the wildcard origin is configured.
func (c CORSConfig) AllowsAnyOrigin() bool {
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" {
			return true
		}
	}
	return false
}

// MinIOConfig addresses the object store. Enabled is false by default: the
// transcoding pipeline still reads and writes the local filesystem, and turning
// object storage on is an explicit opt-in.
type MinIOConfig struct {
	Enabled         bool
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketRaw       string
	BucketProcessed string
	BucketThumbs    string
}

// RateLimitConfig bounds request rates. Enforcement lives in the API process:
// nginx has limit_req rules, but it only ever proxies MinIO, so in the
// documented `make dev` workflow those rules never see an API request.
type RateLimitConfig struct {
	Enabled bool
}

type WorkerConfig struct {
	MaxConcurrentJobs int
	JobTimeout        time.Duration
}

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Storage   StorageConfig
	Auth      AuthConfig
	CORS      CORSConfig
	MinIO     MinIOConfig
	RateLimit RateLimitConfig
	Worker    WorkerConfig
	LogLevel  string
}

// Load reads configuration from the environment, applying defaults, and
// validates the result. A .env file is loaded when present; its absence is not
// an error, but a malformed one is.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

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
			UploadPath:  getEnv("STORAGE_UPLOAD_PATH", "./web/uploads"),
			MaxFileSize: getInt64Env("STORAGE_MAX_FILE_SIZE", 2*1024*1024*1024),
			AllowedFormats: getStringSliceEnv("STORAGE_ALLOWED_FORMATS", []string{
				"video/mp4", "video/mpeg", "video/quicktime", "video/webm", "video/x-matroska",
			}),
			ThumbnailPath:  getEnv("STORAGE_THUMBNAIL_PATH", "./web/uploads/thumbnails"),
			TranscodedPath: getEnv("STORAGE_TRANSCODED_PATH", "./web/uploads/transcoded"),
		},
		Auth: AuthConfig{
			JWTSecret:       getEnv("JWT_SECRET", insecureDefaultJWTSecret),
			JWTIssuer:       getEnv("JWT_ISSUER", "video-streaming-service"),
			AccessTokenTTL:  getDurationEnv("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins: getStringSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:8080"}),
			AllowedMethods: getStringSliceEnv("CORS_ALLOWED_METHODS", []string{
				"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
			}),
			AllowedHeaders: getStringSliceEnv("CORS_ALLOWED_HEADERS", []string{
				"Authorization", "Content-Type", "X-Request-ID",
			}),
			MaxAge: getDurationEnv("CORS_MAX_AGE", 12*time.Hour),
		},
		MinIO: MinIOConfig{
			Enabled:         getBoolEnv("MINIO_ENABLED", false),
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", ""),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", ""),
			UseSSL:          getBoolEnv("MINIO_USE_SSL", false),
			BucketRaw:       getEnv("MINIO_BUCKET_RAW", "videos-raw"),
			BucketProcessed: getEnv("MINIO_BUCKET_PROCESSED", "videos-processed"),
			BucketThumbs:    getEnv("MINIO_BUCKET_THUMBNAILS", "videos-thumbnails"),
		},
		RateLimit: RateLimitConfig{
			Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
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

// Validate checks the configuration for internal consistency. Production is
// held to a stricter standard than development: defaults that are convenient
// locally are refused outright when they would be unsafe in a deployment.
func (c *Config) Validate() error {
	var problems []string

	if c.Server.Port == "" {
		problems = append(problems, "SERVER_PORT is required")
	}
	if c.Database.Host == "" || c.Database.DBName == "" {
		problems = append(problems, "DB_HOST and DB_NAME are required")
	}
	if c.Storage.MaxFileSize <= 0 {
		problems = append(problems, "STORAGE_MAX_FILE_SIZE must be positive")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		problems = append(problems, "DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	if c.Auth.AccessTokenTTL <= 0 {
		problems = append(problems, "JWT_ACCESS_TOKEN_TTL must be positive")
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		problems = append(problems, "CORS_ALLOWED_ORIGINS must list at least one origin")
	}
	if c.MinIO.Enabled && (c.MinIO.AccessKeyID == "" || c.MinIO.SecretAccessKey == "") {
		problems = append(problems, "MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required when MINIO_ENABLED=true")
	}

	if c.Server.IsProduction() {
		if c.Auth.JWTSecret == insecureDefaultJWTSecret {
			problems = append(problems, "JWT_SECRET must be set in production (the default key is public in source control)")
		}
		if len(c.Auth.JWTSecret) < 32 {
			problems = append(problems, "JWT_SECRET must be at least 32 characters in production")
		}
		if c.CORS.AllowsAnyOrigin() {
			problems = append(problems, `CORS_ALLOWED_ORIGINS must not be "*" in production: credentialed requests require an explicit origin allowlist`)
		}
		if c.Database.SSLMode == "disable" {
			problems = append(problems, "DB_SSLMODE must not be 'disable' in production")
		}
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			trimmed = append(trimmed, part)
		}
	}
	if len(trimmed) == 0 {
		return defaultValue
	}
	return trimmed
}
