.PHONY: help dev worker build build-worker test check vet fmt fmt-check lint run \
	migrate-up migrate-down migrate-create templ docker-up docker-down clean \
	install-tools run-all test-player test-hls docker-build docker-up-prod \
	docker-down-prod admin

# Stamped into the binaries via -ldflags -X main.version.
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

help:
	@echo "Available commands:"
	@echo "  make dev           - Run API server with hot reload (Air)"
	@echo "  make worker        - Run worker process"
	@echo "  make build         - Build production binaries"
	@echo "  make build-worker  - Build worker binary"
	@echo "  make run-all       - Start Docker, migrate, API server & worker"
	@echo "  make test          - Run tests with coverage"
	@echo "  make check         - gofmt + vet + test (what CI runs)"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback database migrations"
	@echo "  make migrate-create NAME=xxx - Create new migration"
	@echo "  make templ         - Generate Templ templates"
	@echo "  make docker-up     - Start Docker services (dev infrastructure only)"
	@echo "  make docker-down   - Stop Docker services"
	@echo "  make docker-build  - Build production api and worker images"
	@echo "  make docker-up-prod   - Start the full production stack"
	@echo "  make docker-down-prod - Stop the production stack"
	@echo "  make admin ARGS='...' - Run the operator CLI (e.g. ARGS='promote --username alice --role admin')"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install-tools - Install required tools"
	@echo "  make test-player VIDEO_ID=xxx - Open video player in browser"
	@echo "  make test-hls VIDEO_ID=xxx - Test HLS playback with ffplay"

dev:
	air

worker:
	@go run ./cmd/worker

# Build the package, not a single file: `go build cmd/api/main.go` compiles only
# that one file and fails as soon as the package gains a second.
build:
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api
	@echo "Building worker..."
	@go build -o bin/worker ./cmd/worker
	@echo "Build complete!"

build-worker:
	@echo "Building worker..."
	@go build -o bin/worker ./cmd/worker
	@echo "Worker build complete!"

run-all:
	@echo "Starting all services..."
	@make docker-up
	@make migrate-up
	@echo "Starting API server and worker..."
	@echo "Run 'make dev' in one terminal and 'make worker' in another"

test:
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# What CI should run. Fails on anything gofmt would rewrite, rather than
# silently reformatting it.
check: fmt-check vet test

vet:
	@go vet ./...

fmt-check:
	@test -z "$$(gofmt -l ./cmd ./internal ./pkg)" || \
		(echo "Not gofmt-clean:"; gofmt -l ./cmd ./internal ./pkg; exit 1)

migrate-up:
	@migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/video_streaming?sslmode=disable" up

migrate-down:
	@migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/video_streaming?sslmode=disable" down

migrate-create:
	@migrate create -ext sql -dir migrations -seq $(NAME)

templ:
	@templ generate
	@echo "Templ templates generated!"

docker-up:
	@docker-compose up -d
	@echo "Docker services started!"
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5

docker-down:
	@docker-compose down
	@echo "Docker services stopped!"

docker-build:
	@docker build --target api    --build-arg VERSION=$(VERSION) -t video-streaming-api:$(VERSION)    -t video-streaming-api:latest .
	@docker build --target worker --build-arg VERSION=$(VERSION) -t video-streaming-worker:$(VERSION) -t video-streaming-worker:latest .

# The prod stack reads .env; refuse to start without one rather than let the
# compose interpolation silently fall back to development defaults.
docker-up-prod:
	@test -f .env || (echo ".env not found — copy .env.example and set real secrets first"; exit 1)
	@docker-compose -f docker-compose.prod.yml up -d --build
	@echo "Production stack started"

docker-down-prod:
	@docker-compose -f docker-compose.prod.yml down
	@echo "Production stack stopped"

# Operator CLI. Local: make admin ARGS='create --username alice --email a@b.c --password ...'
# In the prod stack the binary ships inside the api image:
#   docker compose -f docker-compose.prod.yml exec api admin promote --username alice --role admin
admin:
	@go run ./cmd/admin $(ARGS)

clean:
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts!"

install-tools:
	@echo "Installing required tools..."
	# cosmtrek/air was renamed; the old path no longer receives updates.
	@go install github.com/air-verse/air@latest
	# Pinned to the templ version in go.mod: generated code must match the
	# runtime it is compiled against.
	@go install github.com/a-h/templ/cmd/templ@v0.3.977
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed successfully!"

run:
	@go run ./cmd/api

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...
	@gofmt -s -w .

test-player:
ifndef VIDEO_ID
	@echo "Error: VIDEO_ID is required. Usage: make test-player VIDEO_ID=your-video-id"
	@exit 1
endif
	@echo "Opening video player for video $(VIDEO_ID)..."
	@start http://localhost:8080/videos/$(VIDEO_ID)

test-hls:
ifndef VIDEO_ID
	@echo "Error: VIDEO_ID is required. Usage: make test-hls VIDEO_ID=your-video-id"
	@exit 1
endif
	@echo "Testing HLS playback for video $(VIDEO_ID)..."
	@echo "Make sure ffplay (part of FFmpeg) is installed"
	@ffplay -loglevel info http://localhost:8080/api/videos/$(VIDEO_ID)/hls/master.m3u8
