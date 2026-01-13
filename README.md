# Video Streaming Service

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)](https://redis.io/)

A production-ready, high-performance video streaming platform built with Go. Features HTTP Live Streaming (HLS) with adaptive bitrate switching, comprehensive video management, and enterprise-grade scalability using Clean Architecture principles.

## Features

- **HLS Adaptive Streaming** - Multi-quality video delivery (360p, 480p, 720p, 1080p) with automatic bitrate switching
- **Video Processing Pipeline** - Automated transcoding with FFmpeg, thumbnail generation, and background job processing
- **Clean Architecture** - Maintainable, testable codebase with clear separation of concerns
- **Authentication & RBAC** - JWT tokens, bcrypt password hashing, role-based access control
- **Admin Dashboard** - Content moderation, analytics, audit logging, system health monitoring
- **Search & Social** - Full-text search, subscriptions, likes, comments, playlists, notifications
- **Performance Optimized** - Redis caching, database indexing, rate limiting, CDN-ready
- **Monitoring** - Prometheus metrics, Grafana dashboards, health checks

## Tech Stack

| Category | Technology |
|----------|------------|
| **Language** | Go 1.21+ |
| **Framework** | Gin |
| **Database** | PostgreSQL 16 |
| **Cache** | Redis 7 |
| **Object Storage** | MinIO (S3-compatible) |
| **Video Processing** | FFmpeg |
| **Task Queue** | Asynq |
| **Monitoring** | Prometheus, Grafana |
| **Reverse Proxy** | Nginx |

## Architecture

```
cmd/
├── api/                 # HTTP API server
└── worker/              # Background job processor

internal/
├── domain/              # Business entities & errors
├── repository/          # Data access layer (PostgreSQL, Redis)
├── service/             # Business logic
├── handler/             # HTTP handlers
├── middleware/          # Auth, rate limiting, logging
├── queue/               # Background job definitions
└── config/              # Configuration management

pkg/
├── logger/              # Structured logging (Zerolog)
├── jwt/                 # JWT token utilities
├── validator/           # Input validation
├── response/            # API response formatting
└── security/            # Password hashing

migrations/              # Database migrations
queries/                 # SQLC query definitions
web/
├── templates/           # HTML templates
├── static/              # CSS, JS assets
└── uploads/             # Video storage
```

## Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Docker & Docker Compose** - [Download](https://www.docker.com/)
- **FFmpeg** - [Download](https://ffmpeg.org/download.html)
- **Make** - Pre-installed on Linux/Mac, [Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

## Quick Start

```bash
# Clone and navigate to project
cd orchids-video-streaming-foundation

# Copy environment configuration
cp .env.example .env

# Install Go tools (Air, SQLC, golang-migrate)
make install-tools

# Start infrastructure (PostgreSQL, Redis, MinIO)
make docker-up

# Run database migrations
make migrate-up

# Generate type-safe SQL code
make sqlc

# Start development server with hot reload
make dev
```

Server runs at `http://localhost:8080`

## API Reference

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health check (database, Redis status) |
| GET | `/metrics` | Prometheus metrics |

### Video Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/videos` | List videos (paginated) |
| GET | `/api/videos/:id` | Get video details |
| POST | `/api/videos/upload` | Upload new video |
| GET | `/api/videos/:id/status` | Get processing status |
| DELETE | `/api/videos/:id` | Delete video |

### HLS Streaming

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/videos/:id/hls/master.m3u8` | Master playlist (all qualities) |
| GET | `/api/videos/:id/hls/:quality/playlist.m3u8` | Quality-specific playlist |
| GET | `/api/videos/:id/hls/:quality/:segment` | Video segment |
| GET | `/api/videos/:id/stream/:quality` | MP4 fallback |

### Admin Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/admin/videos/:id/retry` | Retry failed processing |
| GET | `/api/admin/queue/stats` | Queue statistics |
| GET | `/api/admin/workers` | Active workers |
| DELETE | `/api/admin/videos/:id/cache` | Clear video cache |

### Web Interface

| Endpoint | Description |
|----------|-------------|
| `/` | Upload page |
| `/videos` | Video list |
| `/videos/:id` | Video player |

## Testing the API

```bash
# Health check
curl http://localhost:8080/health

# List videos
curl http://localhost:8080/api/videos

# Get specific video
curl http://localhost:8080/api/videos/{video-id}

# Upload video
curl -X POST http://localhost:8080/api/videos/upload \
  -F "video=@sample.mp4" \
  -F "title=My Video" \
  -F "description=Description here"

# Check processing status
curl http://localhost:8080/api/videos/{video-id}/status
```

## Make Commands

```bash
make dev           # Development server with hot reload
make build         # Build production binaries
make test          # Run tests with coverage
make migrate-up    # Apply database migrations
make migrate-down  # Rollback migrations
make sqlc          # Generate type-safe SQL
make docker-up     # Start infrastructure
make docker-down   # Stop infrastructure
make clean         # Remove build artifacts
make install-tools # Install Go tools
```

## Configuration

Create `.env` from `.env.example`:

```env
# Server
SERVER_PORT=8080
ENVIRONMENT=development

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=video_streaming
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Storage
UPLOAD_PATH=./web/uploads
MAX_FILE_SIZE=2147483648
ALLOWED_FORMATS=mp4,mov,avi,mkv,webm

# Worker
WORKER_CONCURRENCY=4
JOB_TIMEOUT=3600

# Features
ENABLE_AUTOCOMPLETE=true
MAX_SEARCH_RESULTS=50
SEARCH_CACHE_TTL=300
```

## Database Schema

### Core Tables

| Table | Description |
|-------|-------------|
| `videos` | Video metadata, processing status, HLS paths |
| `users` | User accounts, roles, authentication |
| `video_views` | Real-time view tracking |
| `video_analytics` | Aggregated metrics |

### Social Tables

| Table | Description |
|-------|-------------|
| `subscriptions` | Creator subscriptions |
| `likes` | Video likes/dislikes |
| `comments` | Hierarchical comments |
| `playlists` | User playlists |
| `watch_history` | Viewing history |
| `notifications` | User notifications |

### Admin Tables

| Table | Description |
|-------|-------------|
| `content_reports` | Moderation reports |
| `audit_logs` | Admin action history |

**Key Features:**
- UUID primary keys
- Full-text search with tsvector
- GIN indexes for arrays and search
- Automated triggers for denormalized counts
- Soft deletes for comments

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Database connection failed | Run `make docker-up`, verify with `docker ps` |
| Migration errors | Check version with migrate tool, use `migrate force` if needed |
| Port 8080 in use | Change `SERVER_PORT` in `.env` or kill existing process |
| FFmpeg not found | Install FFmpeg and add to PATH |
| Worker not processing | Verify Redis is running, check worker logs |
| Video upload fails | Check file size limits and allowed formats |

## Production Deployment

### Docker

```bash
# Build production image
docker build -t video-streaming:latest .

# Run with docker-compose
docker-compose -f docker-compose.prod.yml up -d
```

### Infrastructure Requirements

- **PostgreSQL 16+** with connection pooling (PgBouncer recommended)
- **Redis 7+** for caching and sessions
- **MinIO** or S3-compatible storage for videos
- **Nginx** as reverse proxy with caching enabled
- **FFmpeg 6+** with hardware acceleration support

### Monitoring

- Prometheus scrapes `/metrics` endpoint
- Grafana dashboards in `dashboards/` directory
- Health checks via `/health` for load balancers

## Performance

Target metrics:

| Operation | Target (p95) |
|-----------|-------------|
| Homepage load | < 200ms |
| Video list | < 100ms |
| Search | < 300ms |
| Video start | < 500ms |
| API endpoints | < 50ms |
| Cache hit ratio | > 90% |

## License

MIT