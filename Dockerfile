# syntax=docker/dockerfile:1

# One builder, two runtime targets. The images are split because only the
# worker runs ffmpeg: baking a full media toolchain into the API image would
# triple its size and hand every API container an attack surface it never
# uses. Build them with:
#   docker build --target api    -t video-streaming-api .
#   docker build --target worker -t video-streaming-worker .

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.22

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# Modules are downloaded in their own layer so editing source code does not
# re-fetch every dependency on the next build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/api    ./cmd/api \
 && go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/worker ./cmd/worker \
 && go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/admin  ./cmd/admin


FROM alpine:${ALPINE_VERSION} AS api

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api
# The admin CLI ships in the API image so operators can bootstrap the first
# admin account with `docker compose exec api admin create ...` instead of psql.
COPY --from=builder /out/admin /usr/local/bin/admin
# Router serves ./web/static relative to the working directory.
COPY --from=builder /src/web/static ./web/static
# GET /openapi.yaml serves the spec from ./docs; without it the docs page at
# /docs would render but the raw-spec link would 404 in the container.
COPY --from=builder /src/docs/openapi.yaml ./docs/openapi.yaml

# Pre-created and chowned so a named volume mounted here inherits ownership
# the non-root user can write to.
RUN mkdir -p /data/uploads && chown -R app /data/uploads

USER app
EXPOSE 8080

# /health returns 503 while Postgres or Redis is unreachable, so this doubles
# as a readiness signal for depends_on: service_healthy. busybox wget avoids
# installing curl.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["api"]


FROM alpine:${ALPINE_VERSION} AS worker

# ffmpeg brings ffprobe with it; both must be on PATH for transcoding.
RUN apk add --no-cache ca-certificates tzdata ffmpeg \
 && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=builder /out/worker /usr/local/bin/worker

RUN mkdir -p /data/uploads && chown -R app /data/uploads

USER app

# The worker exposes no HTTP endpoint; liveness is the process itself, which
# Docker already tracks, so there is deliberately no HEALTHCHECK here.

ENTRYPOINT ["worker"]
