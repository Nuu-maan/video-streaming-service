# Video Streaming Service

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)](https://redis.io/)

A video streaming platform in Go. Videos are uploaded over HTTP, transcoded to a
multi-quality ladder by a background worker, and served as HLS with an MP4
fallback.

## Architecture

Two binaries share one `internal/` tree. They communicate only through Redis
(via [Asynq](https://github.com/hibiken/asynq)) and a shared PostgreSQL database
— there is no RPC between them.

```
                 ┌──────────┐  enqueue   ┌───────┐  consume  ┌────────────┐
   HTTP ───────► │ cmd/api  │ ─────────► │ Redis │ ────────► │ cmd/worker │
                 └────┬─────┘            └───────┘           └──────┬─────┘
                      │                                             │ ffmpeg
                      └──────────────► PostgreSQL ◄─────────────────┘
```

The upload path: a request lands in `internal/handler`, is validated and written
to disk by `internal/service`, recorded in `videos` with status `uploading`, and
queued. The worker probes it with ffprobe, transcodes to 360/480/720/1080p,
converts each to HLS, writes a master playlist, and marks the video `ready`.

```
cmd/
├── api/                 # HTTP server (thin: builds an app.App and runs it)
└── worker/              # Asynq consumer — transcoding

internal/
├── app/                 # Dependency wiring + router + lifecycle
├── domain/              # Entities, sentinel errors, RBAC roles/permissions
├── repository/          # Persistence contracts (+ postgres/ implementation)
├── service/             # Business logic (auth, upload, transcoding, moderation…)
├── handler/             # HTTP handlers
├── middleware/          # Request ID, logging, CORS, JWT auth, rate limiting
├── queue/               # Asynq task definitions and handlers
├── cache/               # Two-tier cache (in-process L1 → Redis)
├── metrics/             # Prometheus instrumentation
├── storage/             # MinIO client (see "Known gaps")
└── config/              # Environment configuration

pkg/                     # appctx, logger, jwt, security, validator, response
migrations/              # golang-migrate SQL (10 pairs)
web/templates/           # Templ server-rendered pages
```

## Prerequisites

- **Go 1.25+**
- **Docker & Docker Compose** — for PostgreSQL, Redis, MinIO
- **FFmpeg** — `ffmpeg` and `ffprobe` must be on `PATH`
- **Make**

## Quick start

```bash
cp .env.example .env       # defaults work as-is for local development
make install-tools         # air, templ, golang-migrate
make docker-up             # PostgreSQL, Redis, MinIO, Prometheus, Grafana
make migrate-up            # apply schema

make dev                   # terminal 1: API with hot reload
make worker                # terminal 2: transcoding worker
```

The API listens on `http://localhost:8080`. **Both processes are required** — with
no worker running, uploads are accepted but stay in `uploading` forever.

Only infrastructure is containerized. The Go processes run on the host; there is
no `Dockerfile`.

## Authentication

Every write is authenticated with a JWT bearer token; every admin route
additionally requires a permission.

```bash
curl -X POST localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"Str0ng!Passw0rd"}'
# → {"data":{"access_token":"eyJ...","user":{...}}}

curl -X POST localhost:8080/api/videos/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F video=@clip.mp4 -F title='My clip'
```

Roles are `guest`, `user`, `premium`, `moderator`, `admin`; each maps to a
permission set in `internal/domain/role.go`. New accounts get `user`. Promoting
someone to `admin` is a database operation today — there is no endpoint for it.

## API

`✓` = requires a valid bearer token.

### Public

| Method | Endpoint | Auth | Notes |
|---|---|---|---|
| GET | `/health` | | 503 when PostgreSQL or Redis is unreachable |
| GET | `/metrics` | | Prometheus exposition format |
| POST | `/api/auth/register` | | |
| POST | `/api/auth/login` | | Accepts a username or an email |
| POST | `/api/auth/refresh` | | |
| GET | `/api/auth/me` | ✓ | |
| GET | `/api/videos` | optional | Public videos; `?mine=true` with a token lists your own |
| GET | `/api/videos/:id` | optional | |
| GET | `/api/videos/:id/status` | optional | Transcoding progress |

### Streaming

| Method | Endpoint |
|---|---|
| GET | `/api/videos/:id/hls/master.m3u8` |
| GET | `/api/videos/:id/hls/:quality/playlist.m3u8` |
| GET | `/api/videos/:id/hls/:quality/:segment` |
| GET | `/api/videos/:id/stream/:quality` (MP4 fallback, supports `Range`) |

### Authenticated

| Method | Endpoint | Permission |
|---|---|---|
| POST | `/api/videos/upload` | `upload_video` |
| DELETE | `/api/videos/:id` | owner, or `delete_any_video` |
| POST | `/api/reports` | any authenticated user |

### Admin

| Method | Endpoint | Permission |
|---|---|---|
| POST | `/api/admin/videos/:id/retry` | `moderate_content` |
| GET | `/api/admin/queue/stats` | `moderate_content` |
| GET | `/api/admin/workers` | `moderate_content` |
| DELETE | `/api/admin/videos/:id/cache` | `moderate_content` |
| GET | `/api/admin/reports/pending` | `moderate_content` |
| POST | `/api/admin/reports/:id/review` | `moderate_content` |
| POST | `/api/admin/users/:id/ban` · `/unban` | `manage_users` |
| GET | `/api/admin/analytics/dashboard` · `/realtime` · `/top-videos` · `/videos/:id` · `/videos/:id/views` | `view_analytics` |
| GET | `/api/admin/monitoring/metrics` · `/system` · `/queue` · `/database` · `/redis` | `manage_users` |

## Configuration

All configuration is environment-based; see `.env.example`, which lists exactly
the keys `internal/config` reads and nothing else.

Production (`ENVIRONMENT=production`) is validated more strictly and **refuses to
boot** if `JWT_SECRET` is still the development default or shorter than 32
characters, if `CORS_ALLOWED_ORIGINS` is `*`, or if `DB_SSLMODE=disable`.

## Development

```bash
make check     # gofmt + go vet + go test -race   (what CI runs)
make test      # tests with an HTML coverage report
make lint      # golangci-lint
make templ     # regenerate templates after editing web/templates/*.templ
```

## Known gaps

Stated plainly, because the previous README advertised these as done:

- **Object storage is not integrated.** `internal/storage` has a complete MinIO
  client and `MINIO_*` config exists, but the upload, transcoding, and streaming
  paths all still read and write the local filesystem. `MINIO_ENABLED` defaults
  to `false`; turning it on today configures a client that nothing calls.
  Migrating the pipeline to object storage is a real change, not a flag flip.
- **Social features are schema-only.** Migration 9 creates `subscriptions`,
  `likes`, `comments`, `playlists`, `watch_history`, and `notifications`. No
  endpoint touches any of them.
- **nginx does not proxy the API.** `nginx.conf` fronts MinIO only. Its
  `limit_req` rules therefore never see an API request; rate limiting is enforced
  in-process by `internal/middleware` instead.
- **No `Dockerfile` / `docker-compose.prod.yml`.** `docker-compose.yml` brings up
  infrastructure only.
- **Email verification and password reset** have columns, domain methods, and
  token generation, but no delivery mechanism and no endpoints.
- `sqlc.yaml` and `queries/` are vestigial — every repository is hand-written SQL
  and no generated code is committed or imported.

## License

MIT.
