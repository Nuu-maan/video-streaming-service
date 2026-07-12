package config

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// productionSecret is long enough to satisfy the 32-character minimum and is
// not the insecure default.
const productionSecret = "a-real-production-secret-of-sufficient-length"

// validConfig returns a configuration that Validate accepts in development.
// Individual tests mutate one field at a time.
func validConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        "8080",
			Environment: "development",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         "5432",
			User:         "postgres",
			Password:     "postgres",
			DBName:       "video_streaming",
			SSLMode:      "disable",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Storage: StorageConfig{
			MaxFileSize: 2 * 1024 * 1024 * 1024,
		},
		Auth: AuthConfig{
			JWTSecret:      insecureDefaultJWTSecret,
			JWTIssuer:      "video-streaming-service",
			AccessTokenTTL: 15 * time.Minute,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"http://localhost:8080"},
		},
	}
}

// productionConfig returns a configuration that Validate accepts in production.
func productionConfig() *Config {
	cfg := validConfig()
	cfg.Server.Environment = EnvProduction
	cfg.Auth.JWTSecret = productionSecret
	cfg.Database.SSLMode = "require"
	cfg.CORS.AllowedOrigins = []string{"https://videos.example.com"}
	return cfg
}

func TestDatabaseConfigDSN(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		password string
		dbName   string
		sslMode  string
	}{
		{
			// Regression: the old keyword/value DSN left the password unescaped,
			// so anything with a space, '@', ':' or '/' produced a broken or
			// misparsed connection string.
			name:     "password with special characters",
			user:     "postgres",
			password: `p@ss w:rd/!`,
			dbName:   "video_streaming",
			sslMode:  "disable",
		},
		{
			name:     "password with quotes and backslashes",
			user:     "postgres",
			password: `it's "quoted"\and\escaped`,
			dbName:   "video_streaming",
			sslMode:  "require",
		},
		{
			name:     "password with percent and plus",
			user:     "app_user",
			password: "100%+more?#fragmenty",
			dbName:   "video_streaming",
			sslMode:  "verify-full",
		},
		{
			name:     "plain password",
			user:     "postgres",
			password: "postgres",
			dbName:   "video_streaming",
			sslMode:  "disable",
		},
		{
			name:     "empty password",
			user:     "postgres",
			password: "",
			dbName:   "video_streaming",
			sslMode:  "disable",
		},
		{
			name:     "user with special characters",
			user:     "user@corp:group",
			password: "s3cret",
			dbName:   "video_streaming",
			sslMode:  "disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DatabaseConfig{
				Host:     "db.internal",
				Port:     "5432",
				User:     tt.user,
				Password: tt.password,
				DBName:   tt.dbName,
				SSLMode:  tt.sslMode,
			}

			dsn := cfg.DSN()

			parsed, err := url.Parse(dsn)
			if err != nil {
				t.Fatalf("url.Parse(%q): %v", dsn, err)
			}

			if parsed.Scheme != "postgres" {
				t.Errorf("scheme = %q, want %q", parsed.Scheme, "postgres")
			}
			if parsed.Host != "db.internal:5432" {
				t.Errorf("host = %q, want %q", parsed.Host, "db.internal:5432")
			}
			if got := strings.TrimPrefix(parsed.Path, "/"); got != tt.dbName {
				t.Errorf("dbname = %q, want %q", got, tt.dbName)
			}
			if parsed.User == nil {
				t.Fatal("DSN carries no userinfo")
			}
			if got := parsed.User.Username(); got != tt.user {
				t.Errorf("user = %q, want %q", got, tt.user)
			}

			gotPassword, hasPassword := parsed.User.Password()
			if !hasPassword {
				t.Fatalf("DSN carries no password; DSN = %q", dsn)
			}
			if gotPassword != tt.password {
				t.Errorf("password = %q, want %q (DSN = %q)", gotPassword, tt.password, dsn)
			}

			if got := parsed.Query().Get("sslmode"); got != tt.sslMode {
				t.Errorf("sslmode = %q, want %q", got, tt.sslMode)
			}

			// The raw DSN must not leak the unescaped password: if it did,
			// the delimiters would be reparsed as URL structure.
			if strings.Contains(tt.password, " ") && strings.Contains(dsn, " ") {
				t.Errorf("DSN %q contains a raw space; the password is not encoded", dsn)
			}
		})
	}
}

func TestConfigValidateDevelopment(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; empty means the config must be accepted
	}{
		{
			name:   "defaults are accepted in development",
			mutate: func(*Config) {},
		},
		{
			name: "insecure default JWT secret is fine in development",
			mutate: func(c *Config) {
				c.Auth.JWTSecret = insecureDefaultJWTSecret
			},
		},
		{
			name: "wildcard CORS is fine in development",
			mutate: func(c *Config) {
				c.CORS.AllowedOrigins = []string{"*"}
			},
		},
		{
			name: "sslmode disable is fine in development",
			mutate: func(c *Config) {
				c.Database.SSLMode = "disable"
			},
		},
		{
			name:    "empty port rejected",
			mutate:  func(c *Config) { c.Server.Port = "" },
			wantErr: "SERVER_PORT",
		},
		{
			name:    "empty db host rejected",
			mutate:  func(c *Config) { c.Database.Host = "" },
			wantErr: "DB_HOST",
		},
		{
			name:    "empty db name rejected",
			mutate:  func(c *Config) { c.Database.DBName = "" },
			wantErr: "DB_NAME",
		},
		{
			name:    "non-positive max file size rejected",
			mutate:  func(c *Config) { c.Storage.MaxFileSize = 0 },
			wantErr: "STORAGE_MAX_FILE_SIZE",
		},
		{
			name:    "idle conns above open conns rejected",
			mutate:  func(c *Config) { c.Database.MaxIdleConns = 50 },
			wantErr: "DB_MAX_IDLE_CONNS",
		},
		{
			name:    "non-positive access token TTL rejected",
			mutate:  func(c *Config) { c.Auth.AccessTokenTTL = 0 },
			wantErr: "JWT_ACCESS_TOKEN_TTL",
		},
		{
			name:    "empty CORS origins rejected",
			mutate:  func(c *Config) { c.CORS.AllowedOrigins = nil },
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "MinIO enabled without credentials rejected",
			mutate: func(c *Config) {
				c.MinIO.Enabled = true
				c.MinIO.AccessKeyID = ""
				c.MinIO.SecretAccessKey = ""
			},
			wantErr: "MINIO_ACCESS_KEY",
		},
		{
			name: "MinIO enabled with credentials accepted",
			mutate: func(c *Config) {
				c.MinIO.Enabled = true
				c.MinIO.AccessKeyID = "minioadmin"
				c.MinIO.SecretAccessKey = "minioadmin"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want an error mentioning %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidateProduction(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; empty means the config must be accepted
	}{
		{
			name:   "properly configured production config accepted",
			mutate: func(*Config) {},
		},
		{
			name:    "insecure default JWT secret rejected",
			mutate:  func(c *Config) { c.Auth.JWTSecret = insecureDefaultJWTSecret },
			wantErr: "JWT_SECRET must be set in production",
		},
		{
			name:    "short JWT secret rejected",
			mutate:  func(c *Config) { c.Auth.JWTSecret = strings.Repeat("s", 31) },
			wantErr: "JWT_SECRET must be at least 32 characters",
		},
		{
			name:   "JWT secret of exactly 32 characters accepted",
			mutate: func(c *Config) { c.Auth.JWTSecret = strings.Repeat("s", 32) },
		},
		{
			name:    "wildcard CORS origin rejected",
			mutate:  func(c *Config) { c.CORS.AllowedOrigins = []string{"*"} },
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "wildcard alongside explicit origins still rejected",
			mutate: func(c *Config) {
				c.CORS.AllowedOrigins = []string{"https://videos.example.com", "*"}
			},
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name:    "sslmode disable rejected",
			mutate:  func(c *Config) { c.Database.SSLMode = "disable" },
			wantErr: "DB_SSLMODE",
		},
		{
			name:   "sslmode verify-full accepted",
			mutate: func(c *Config) { c.Database.SSLMode = "verify-full" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := productionConfig()
			tt.mutate(cfg)

			if !cfg.Server.IsProduction() {
				t.Fatal("test setup is not a production config")
			}

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want an error mentioning %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestServerConfigHelpers(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		wantProd    bool
		host, port  string
		wantAddress string
	}{
		{name: "production", environment: "production", wantProd: true, host: "0.0.0.0", port: "8080", wantAddress: "0.0.0.0:8080"},
		{name: "development", environment: "development", wantProd: false, host: "127.0.0.1", port: "3000", wantAddress: "127.0.0.1:3000"},
		{name: "staging is not production", environment: "staging", wantProd: false, host: "localhost", port: "80", wantAddress: "localhost:80"},
		{name: "case sensitive", environment: "Production", wantProd: false, host: "localhost", port: "80", wantAddress: "localhost:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ServerConfig{Environment: tt.environment, Host: tt.host, Port: tt.port}
			if got := c.IsProduction(); got != tt.wantProd {
				t.Errorf("IsProduction() = %v, want %v", got, tt.wantProd)
			}
			if got := c.Address(); got != tt.wantAddress {
				t.Errorf("Address() = %q, want %q", got, tt.wantAddress)
			}
		})
	}
}

func TestCORSConfigAllowsOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		want           bool
	}{
		{
			name:           "exact match",
			allowedOrigins: []string{"https://videos.example.com"},
			origin:         "https://videos.example.com",
			want:           true,
		},
		{
			name:           "case-insensitive match",
			allowedOrigins: []string{"https://Videos.Example.com"},
			origin:         "https://videos.example.com",
			want:           true,
		},
		{
			name:           "match in a longer list",
			allowedOrigins: []string{"https://a.example.com", "https://b.example.com"},
			origin:         "https://b.example.com",
			want:           true,
		},
		{
			name:           "wildcard allows anything",
			allowedOrigins: []string{"*"},
			origin:         "https://evil.example.com",
			want:           true,
		},
		{
			name:           "no match",
			allowedOrigins: []string{"https://videos.example.com"},
			origin:         "https://evil.example.com",
			want:           false,
		},
		{
			name:           "different scheme does not match",
			allowedOrigins: []string{"https://videos.example.com"},
			origin:         "http://videos.example.com",
			want:           false,
		},
		{
			name:           "empty allowlist denies",
			allowedOrigins: nil,
			origin:         "https://videos.example.com",
			want:           false,
		},
		{
			name:           "empty origin denied unless listed",
			allowedOrigins: []string{"https://videos.example.com"},
			origin:         "",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CORSConfig{AllowedOrigins: tt.allowedOrigins}
			if got := c.AllowsOrigin(tt.origin); got != tt.want {
				t.Errorf("AllowsOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCORSConfigAllowsAnyOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		want           bool
	}{
		{name: "wildcard alone", allowedOrigins: []string{"*"}, want: true},
		{name: "wildcard among others", allowedOrigins: []string{"https://a.example.com", "*"}, want: true},
		{name: "explicit origins only", allowedOrigins: []string{"https://a.example.com"}, want: false},
		{name: "empty", allowedOrigins: nil, want: false},
		{name: "wildcard-looking but not wildcard", allowedOrigins: []string{"https://*.example.com"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CORSConfig{AllowedOrigins: tt.allowedOrigins}
			if got := c.AllowsAnyOrigin(); got != tt.want {
				t.Errorf("AllowsAnyOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisConfigAddress(t *testing.T) {
	c := RedisConfig{Host: "redis.internal", Port: "6379"}
	if got, want := c.Address(), "redis.internal:6379"; got != want {
		t.Errorf("Address() = %q, want %q", got, want)
	}
}

func TestLoad(t *testing.T) {
	t.Run("defaults produce a valid development config", func(t *testing.T) {
		// Pin the environment so an ambient value cannot flip us into
		// production validation.
		t.Setenv("ENVIRONMENT", "development")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.Server.Port != "8080" {
			t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "8080")
		}
		if cfg.Auth.JWTSecret != insecureDefaultJWTSecret {
			t.Errorf("Auth.JWTSecret = %q, want the insecure development default", cfg.Auth.JWTSecret)
		}
		if cfg.Auth.AccessTokenTTL != 15*time.Minute {
			t.Errorf("Auth.AccessTokenTTL = %v, want 15m", cfg.Auth.AccessTokenTTL)
		}
		if cfg.Server.IsProduction() {
			t.Error("IsProduction() = true, want false")
		}
	})

	t.Run("environment overrides are applied", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "development")
		t.Setenv("SERVER_PORT", "9999")
		t.Setenv("DB_PASSWORD", `p@ss w:rd/!`)
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "45m")
		t.Setenv("DB_MAX_OPEN_CONNS", "42")
		t.Setenv("RATE_LIMIT_ENABLED", "false")
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.Server.Port != "9999" {
			t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9999")
		}
		if cfg.Database.Password != `p@ss w:rd/!` {
			t.Errorf("Database.Password = %q, want the raw value", cfg.Database.Password)
		}
		if cfg.Auth.AccessTokenTTL != 45*time.Minute {
			t.Errorf("Auth.AccessTokenTTL = %v, want 45m", cfg.Auth.AccessTokenTTL)
		}
		if cfg.Database.MaxOpenConns != 42 {
			t.Errorf("Database.MaxOpenConns = %d, want 42", cfg.Database.MaxOpenConns)
		}
		if cfg.RateLimit.Enabled {
			t.Error("RateLimit.Enabled = true, want false")
		}
		want := []string{"https://a.example.com", "https://b.example.com"}
		if len(cfg.CORS.AllowedOrigins) != len(want) {
			t.Fatalf("CORS.AllowedOrigins = %v, want %v", cfg.CORS.AllowedOrigins, want)
		}
		for i := range want {
			if cfg.CORS.AllowedOrigins[i] != want[i] {
				t.Errorf("CORS.AllowedOrigins[%d] = %q, want %q", i, cfg.CORS.AllowedOrigins[i], want[i])
			}
		}
	})

	t.Run("production without a JWT secret fails to load", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("DB_SSLMODE", "require")
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://videos.example.com")

		cfg, err := Load()
		if err == nil {
			t.Fatalf("Load() = %+v, want an error", cfg)
		}
		if !strings.Contains(err.Error(), "JWT_SECRET") {
			t.Errorf("Load() error = %q, want it to mention JWT_SECRET", err)
		}
	})

	t.Run("fully configured production loads", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("JWT_SECRET", productionSecret)
		t.Setenv("DB_SSLMODE", "require")
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://videos.example.com")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if !cfg.Server.IsProduction() {
			t.Error("IsProduction() = false, want true")
		}
		if cfg.CORS.AllowsAnyOrigin() {
			t.Error("production config allows any origin")
		}
		if !cfg.CORS.AllowsOrigin("https://videos.example.com") {
			t.Error("configured origin is not allowed")
		}
	})
}
