# Video Streaming Service - Professional Foundation

A production-ready video streaming backend service built with Go, following Clean Architecture principles. This is Phase 1 - the foundation for a scalable video platform with PostgreSQL, Redis, and modern Go best practices.

## 🎯 What We Built

This is a **professionally structured Go application** that follows industry standards and Clean Architecture. Think of it as the solid foundation of a house - everything is organized, tested, and ready for features to be added on top.

## 🏗️ Architecture Overview

### Clean Architecture Layers

```
cmd/                    # Application entry points
├── api/               # HTTP API server
└── worker/            # Background job processor

internal/              # Private application code
├── domain/           # Business entities (Video, errors)
├── repository/       # Database access layer
│   ├── postgres/    # PostgreSQL implementation
│   └── redis/       # Redis implementation
├── service/         # Business logic layer
├── handler/         # HTTP handlers (controllers)
├── middleware/      # HTTP middleware
├── worker/          # Background job handlers
└── config/          # Configuration management

pkg/                   # Public reusable packages
├── logger/          # Structured logging
├── validator/       # Input validation
└── response/        # API response formatters

migrations/           # Database migrations (version control for DB)
queries/             # SQL queries for SQLC
web/                 # Frontend assets
├── templates/      # HTML templates
├── static/         # CSS, JS, images
└── uploads/        # Uploaded videos
```

### Why This Structure?

- **Separation of Concerns**: Each layer has one responsibility
- **Testability**: Easy to mock dependencies and write tests
- **Maintainability**: Changes in one layer don't break others
- **Scalability**: Can swap implementations (e.g., PostgreSQL → MySQL)

## 🚀 Tech Stack

| Technology | Purpose | Why? |
|------------|---------|------|
| **Go** | Backend language | Fast, compiled, great for APIs |
| **Gin** | HTTP framework | Fastest Go web framework |
| **PostgreSQL** | Main database | Reliable, ACID-compliant, full-featured |
| **Redis** | Cache & sessions | Super fast in-memory data store |
| **SQLC** | Type-safe SQL | Generates Go code from SQL queries |
| **golang-migrate** | Database migrations | Version control for database schema |
| **Air** | Hot reload | Auto-restart on code changes |
| **Templ** | HTML templates | Type-safe templates (compile-time checks) |
| **HTMX** | Dynamic UI | No JavaScript framework needed |
| **Zerolog** | Structured logging | Fast, structured JSON logs |

## 📋 Prerequisites

Before running this project, install:

- **Go 1.21+** ([download](https://golang.org/dl/))
- **Docker & Docker Compose** ([download](https://www.docker.com/))
- **Make** (usually pre-installed on Linux/Mac, [Windows setup](https://gnuwin32.sourceforge.net/packages/make.htm))

Optional (will be auto-installed via Makefile):
- Air, SQLC, Templ, golang-migrate

## 🎬 Quick Start

### 1. Clone and Setup

```bash
# Navigate to project directory
cd orchids-video-streaming-foundation

# Copy environment variables
cp .env.example .env

# Install required Go tools
make install-tools

# Start PostgreSQL and Redis with Docker
make docker-up
```

### 2. Run Database Migrations

```bash
# Create tables in PostgreSQL
make migrate-up
```

### 3. Generate SQLC Code

```bash
# Generate type-safe Go code from SQL queries
make sqlc
```

### 4. Run the Server

```bash
# Development mode with hot reload
make dev

# Or run directly
make run
```

The server will start at `http://localhost:8080`

### 5. Test the API

```bash
# Health check
curl http://localhost:8080/health

# Metrics
curl http://localhost:8080/metrics

# API endpoints (placeholders for now)
curl http://localhost:8080/api/v1/videos
```

## 🛠️ Available Commands

Run `make help` to see all commands:

```bash
make dev           # Run with hot reload (Air)
make build         # Build production binaries
make test          # Run tests with coverage
make migrate-up    # Run database migrations
make migrate-down  # Rollback migrations
make sqlc          # Generate SQLC code
make templ         # Generate Templ templates
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make clean         # Remove build artifacts
make install-tools # Install required tools
```

## 📁 Project Structure Explained

### Domain Layer (`internal/domain/`)

**Pure business logic** - no external dependencies.

- `video.go`: Video entity with business methods
  - `NewVideo()`: Creates a new video with validation
  - `IsProcessing()`: Checks if video is being processed
  - `MarkAsReady()`: Updates video status to ready
  - `Validate()`: Business validation rules

- `errors.go`: Custom error types
  - `ErrVideoNotFound`
  - `ErrInvalidTitle`
  - `ErrProcessingFailed`

### Repository Layer (`internal/repository/`)

**Database access** - all SQL queries live here.

- `interfaces.go`: Defines contracts (interfaces)
- `postgres/`: PostgreSQL implementation using SQLC
  - Type-safe queries (no SQL injection)
  - Connection pooling for performance
  - Context support for timeouts

### Configuration (`internal/config/`)

**Environment management** - loads settings from `.env`

- Server config (host, port, timeouts)
- Database config (connection pool settings)
- Redis config
- Storage config (file size limits, paths)
- Worker config (concurrency limits)

### Logger (`pkg/logger/`)

**Structured logging** with zerolog:

- Pretty console output in development
- JSON logs in production (for log aggregation)
- Request ID tracking
- Context-aware logging

## 🔌 API Endpoints

### Health & Monitoring

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (DB, Redis status) |
| GET | `/metrics` | Service metrics |

### Videos API (v1)

| Method | Path | Description | Status |
|--------|------|-------------|--------|
| GET | `/api/v1/videos` | List all videos | 🚧 Placeholder |
| GET | `/api/v1/videos/:id` | Get video by ID | 🚧 Placeholder |
| POST | `/api/v1/videos` | Upload new video | 🚧 Placeholder |
| DELETE | `/api/v1/videos/:id` | Delete video | 🚧 Placeholder |

## 🗄️ Database Schema

### Videos Table

```sql
CREATE TABLE videos (
    id                   UUID PRIMARY KEY,
    title                VARCHAR(255) NOT NULL,
    description          TEXT,
    filename             VARCHAR(500) NOT NULL,
    file_path            VARCHAR(1000) NOT NULL,
    file_size            BIGINT CHECK (file_size > 0),
    duration             INTEGER DEFAULT 0,
    status               video_status NOT NULL,
    mime_type            VARCHAR(100) NOT NULL,
    original_resolution  VARCHAR(50),
    thumbnail_path       VARCHAR(1000),
    transcoding_progress INTEGER CHECK (0-100),
    available_qualities  TEXT[],
    created_at           TIMESTAMP WITH TIME ZONE,
    updated_at           TIMESTAMP WITH TIME ZONE,
    processed_at         TIMESTAMP WITH TIME ZONE
);
```

**Indexes** for performance:
- `status` (for filtering by status)
- `created_at` (for sorting by date)
- Full-text search on `title` and `description`

**Automatic Triggers**:
- `updated_at` automatically updates on every change

## 🔐 Environment Variables

All configuration is in `.env` file. Key settings:

```bash
# Server
SERVER_PORT=8080              # HTTP port
ENVIRONMENT=development       # development/production

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=video_streaming
DB_MAX_OPEN_CONNS=25         # Connection pool size

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Storage
STORAGE_MAX_FILE_SIZE=2GB    # Max upload size
STORAGE_UPLOAD_PATH=./web/uploads

# Logging
LOG_LEVEL=info               # debug/info/warn/error
```

## 🧪 Testing

```bash
# Run all tests with race detector
make test

# This generates:
# - coverage.out (coverage data)
# - coverage.html (visual coverage report)
```

### Testing Approach

- **Unit tests**: Services with mocked repositories
- **Integration tests**: Repositories with real database
- **Handler tests**: HTTP endpoints with httptest

## 🏭 Best Practices Implemented

### Error Handling

```go
// Service returns domain errors
if video == nil {
    return domain.ErrVideoNotFound
}

// Handler translates to HTTP status
if errors.Is(err, domain.ErrVideoNotFound) {
    c.JSON(404, gin.H{"error": "Video not found"})
}
```

### Context Usage

Every function takes `context.Context` as first parameter:
- Cancellation support
- Timeout handling
- Request ID propagation

### Logging Pattern

```go
log.Info(ctx, "Video created", map[string]interface{}{
    "video_id": video.ID,
    "title": video.Title,
})
```

### Dependency Injection

No global variables (except config). Pass dependencies as parameters:

```go
type VideoService struct {
    repo repository.VideoRepository
    log  *logger.Logger
}

func NewVideoService(repo repository.VideoRepository, log *logger.Logger) *VideoService {
    return &VideoService{repo: repo, log: log}
}
```

## 🚦 Production Readiness Checklist

- ✅ Clean Architecture (maintainable, testable)
- ✅ Configuration management (12-factor app)
- ✅ Structured logging (for monitoring)
- ✅ Database migrations (version control)
- ✅ Connection pooling (performance)
- ✅ Graceful shutdown (no dropped requests)
- ✅ Health checks (for load balancers)
- ✅ Request ID tracking (for debugging)
- ✅ CORS support (for frontend)
- ✅ Error handling (consistent responses)
- ✅ Type-safe SQL (SQLC prevents SQL injection)

## 🎓 Learning Resources

### Go Concepts Used

- **Interfaces**: For dependency injection and mocking
- **Context**: For cancellation and request scoping
- **Channels**: For graceful shutdown
- **Goroutines**: For concurrent server startup
- **Struct embedding**: For middleware composition
- **Error wrapping**: With `fmt.Errorf` and `%w`

### Architecture Patterns

- **Repository Pattern**: Abstracts data access
- **Service Layer**: Contains business logic
- **Middleware Pattern**: Request processing pipeline
- **Factory Pattern**: `NewVideo()`, `NewService()`

## 🔜 Next Steps (Phase 2)

Now that the foundation is ready:

1. **Video Upload Handler**: Multipart file upload
2. **Service Layer**: Business logic for video operations
3. **Video Processing Worker**: Transcoding with FFmpeg
4. **Storage Service**: File system operations
5. **Streaming Endpoints**: HLS/DASH video delivery
6. **User Authentication**: JWT or session-based
7. **Rate Limiting**: Prevent abuse
8. **Video Analytics**: View counts, watch time

## 🐛 Troubleshooting

### Database Connection Failed

```bash
# Check if PostgreSQL is running
make docker-up

# Verify connection
docker exec -it video_streaming_postgres psql -U postgres -d video_streaming
```

### Migration Errors

```bash
# Check migration status
migrate -path migrations -database "postgres://..." version

# Force fix version (if needed)
migrate -path migrations -database "postgres://..." force VERSION
```

### Port Already in Use

```bash
# Change port in .env
SERVER_PORT=8081

# Or kill process using port
# Windows: netstat -ano | findstr :8080
# Linux/Mac: lsof -ti:8080 | xargs kill
```

## 📞 Support

For questions or issues:
1. Check this README thoroughly
2. Review code comments
3. Check existing issues/discussions

## 📄 License

This project is part of the Orchids video streaming tutorial series.

---

**Built with ❤️ using Go and Clean Architecture principles**
