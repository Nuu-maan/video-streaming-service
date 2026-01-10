# Video Streaming Service

A production-ready video streaming backend built with Go, implementing HTTP Live Streaming (HLS) with adaptive bitrate switching and comprehensive video management. The service uses Clean Architecture principles with PostgreSQL, Redis, and FFmpeg for professional video processing.

## Current Status

The platform has completed six major development phases:

**Phase 1: Core Infrastructure**
Built the foundation with Clean Architecture, PostgreSQL database, Redis caching, structured logging, and configuration management. Established the repository pattern and service layer architecture.

**Phase 2: Video Upload Pipeline**
Implemented multipart file uploads with chunking support, input validation, file system management, metadata extraction, and temporary storage handling. Added comprehensive error handling and upload progress tracking.

**Phase 3: Video Processing & Transcoding**
Integrated FFmpeg for video transcoding with multiple quality outputs (360p, 480p, 720p, 1080p). Created background job processing using Asynq, thumbnail generation, and asynchronous task queues. Implemented process management and resource optimization.

**Phase 4: HLS Adaptive Streaming**
Added HTTP Live Streaming protocol support with master playlist generation, quality-specific playlists, and segment-based delivery. Implemented Video.js player with automatic quality switching, Redis playlist caching, and MP4 fallback for broader compatibility. CORS-enabled streaming with proper cache headers.

**Phase 5: Authentication & Authorization**
Completed user management system with bcrypt password hashing, JWT tokens, session management, and role-based access control (user, premium, moderator, admin). Includes middleware for authentication, authorization, and rate limiting.

**Phase 6: Admin Dashboard & System Monitoring**
Implemented comprehensive admin dashboard with analytics tracking, content moderation system, audit logging, and system health monitoring. Features include real-time view tracking, content reports, user ban management, and performance metrics monitoring (CPU, memory, database, Redis, queue).

## What This Platform Does

This is a professional-grade video streaming service that handles the complete video lifecycle from upload to delivery. Users can upload videos that get automatically processed into multiple quality levels, then streamed efficiently using industry-standard HLS protocol with adaptive bitrate switching.

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

**Technology Stack**

**Backend Framework**
- Go 1.21+ for performance and concurrency
- Gin web framework for HTTP routing
- Clean Architecture for maintainability

**Data Storage**
- PostgreSQL for relational data with ACID guarantees
- Redis for session management, caching, and real-time view tracking
- File system for video storage

**Video Processing**
- FFmpeg for transcoding and thumbnail generation
- HLS protocol for adaptive streaming
- Multiple quality outputs (360p to 1080p)

**Background Jobs**
- Asynq for distributed task queue
- Worker processes for async video processing

**Development Tools**
- SQLC for type-safe SQL queries
- Templ for HTML templates
- Air for hot reload development
- golang-migrate for database versioning
- Zerolog for structured logging

**Authentication & Security**
- Bcrypt for password hashing
- JWT for API tokens
- Redis sessions for web authentication
- Role-based access control (RBAC)

**Monitoring & Analytics**
- gopsutil for system metrics (CPU, memory, disk)
- Custom analytics tracking with Redis caching
- Audit logging for admin actions
- Content moderation system

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

## Project Structure

The codebase follows Clean Architecture with clear separation of concerns:

**Domain Layer** (`internal/domain/`)
Core business entities and logic. Includes Video, User, Analytics, Report, AuditLog, and System models with validation methods, error definitions, and role-based permission system.

**Repository Layer** (`internal/repository/`)
Database access using SQLC for type-safe queries. PostgreSQL implementations for videos, users, analytics, content reports, and audit logs with connection pooling and transaction support.

**Service Layer** (`internal/service/`)
Business logic including upload handling, FFmpeg transcoding, session management, password security with bcrypt hashing, analytics tracking, content moderation, audit logging, and system monitoring.

**Handler Layer** (`internal/handler/`)
HTTP request handling with Gin framework. Includes upload endpoints, streaming API, authentication, admin functions, moderation tools, and page rendering.

**Queue Layer** (`internal/queue/`)
Background job processing with Asynq for video transcoding and thumbnail generation.

**Configuration** (`internal/config/`)
Environment-based settings for server, database, Redis, storage, and worker processes.

**Utilities** (`pkg/`)
Reusable packages for logging (zerolog), JWT tokens, password security, response formatting, and validation.

## API Endpoints

**Health & Monitoring**
- GET `/health` - Service health check (database and Redis status)
- GET `/metrics` - Performance and usage metrics

**Video Management**
- GET `/api/v1/videos` - List all videos
- GET `/api/v1/videos/:id` - Get specific video details
- POST `/api/v1/videos/upload` - Upload new video
- DELETE `/api/v1/videos/:id` - Delete video

**HLS Streaming**
- GET `/api/videos/:id/hls/master.m3u8` - Master playlist with all quality levels
- GET `/api/videos/:id/hls/:quality/playlist.m3u8` - Quality-specific playlist
- GET `/api/videos/:id/hls/:quality/:segment` - Video segment delivery
- GET `/api/videos/:id/stream/:quality` - MP4 fallback streaming

**Admin Operations**
- DELETE `/api/admin/videos/:id/cache` - Clear playlist cache for video

**Web Interface**
- GET `/` - Video list page
- GET `/upload` - Upload page
- GET `/videos/:id` - Video player page

**Database Schema**

**Videos Table**
Stores video metadata, processing status, and file references. Includes fields for HLS support (master playlist path, quality variants), transcoding progress, and status tracking (uploaded, processing, ready, failed).

**Users Table**
User accounts with authentication fields (password hash, email verification), profile data (username, bio, avatar), role-based permissions (user, premium, moderator, admin), ban management fields, and OAuth integration support.

**Analytics Tables**
- `video_views` - Real-time view tracking with user and timestamp data
- `video_analytics` - Aggregated video performance metrics (total views, unique viewers, watch time)
- `user_analytics` - User engagement metrics (uploads, views, activity patterns)

**Moderation Tables**
- `content_reports` - User-submitted content reports with status tracking (pending, reviewed, resolved)
- `audit_logs` - Complete audit trail of admin actions with metadata

**Key Features**
- UUID primary keys
- Automatic timestamp management
- Full-text search on video titles and descriptions
- Indexes on frequently queried fields
- Foreign key relationships with cascade deletes
- Check constraints for data validation

## Environment Configuration

Configuration is managed through `.env` file with these key settings:

**Server Settings**
- `SERVER_PORT` - HTTP port (default: 8080)
- `ENVIRONMENT` - development or production mode
- Timeout configurations for reads, writes, and graceful shutdown

**Database**
- PostgreSQL connection details (host, port, credentials)
- Connection pool settings for performance optimization
- Max open connections and idle connection limits

**Redis**
- Connection configuration for caching and sessions
- Pool size and connection management

**Storage**
- Upload path for video files
- Maximum file size limits (default: 2GB)
- Paths for thumbnails and transcoded outputs
- Allowed video formats

**Worker Configuration**
- Concurrent job processing limits
- Job timeout settings for transcoding operations

**Logging**
- Log level control (debug, info, warn, error)
- Structured JSON output for production

## Testing

Run tests with coverage analysis:

```bash
make test
```

This executes all tests with race detection and generates coverage reports (coverage.out and coverage.html). The testing approach includes unit tests for services with mocked dependencies, integration tests for repositories with real database connections, and handler tests for HTTP endpoints using httptest.

## Implementation Details

**Error Handling**
Domain-specific errors propagate through layers and get translated to appropriate HTTP status codes at the handler level. Uses error wrapping for context preservation.

**Context Usage**
Every function accepts context.Context for proper cancellation, timeout handling, and request ID propagation across the application.

**Logging Pattern**
Structured logging with contextual fields for debugging and monitoring. Request IDs track operations across services.

**Dependency Injection**
No global state. Dependencies passed as parameters to constructors, enabling easy testing and flexibility.

**Production Readiness**
- Clean Architecture for maintainable code
- Configuration management following 12-factor principles
- Graceful shutdown without dropping requests
- Connection pooling for database and Redis
- Health checks for load balancer integration
- Type-safe SQL preventing injection attacks
- Request ID tracking for distributed tracing

## Troubleshooting

**Database Connection Issues**
Start PostgreSQL with `make docker-up` and verify connection using `docker exec -it video_streaming_postgres psql -U postgres -d video_streaming`

**Migration Errors**
Check migration status with migrate tool and use force command if needed to fix version conflicts.

**Port Conflicts**
Change SERVER_PORT in .env file or terminate the process using the port.

**FFmpeg Not Found**
Install FFmpeg and ensure it's in system PATH for video processing to work.

**Worker Not Processing Jobs**
Verify Redis is running and check worker logs for errors. Ensure Asynq is properly configured.

## Next Development Phases

**Future Enhancements**
- Admin dashboard UI templates (HTMX/Templ)
- Admin API handlers with authentication middleware
- Video analytics visualization
- Content moderation queue interface
- Advanced search and filtering
- Comment system and social features
- Content recommendation engine
- CDN integration for global delivery
- Live streaming capabilities
- Advanced notification system

## License

This project is part of the Orchids video streaming platform series.
