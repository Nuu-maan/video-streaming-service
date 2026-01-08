.PHONY: help dev worker build build-worker test migrate-up migrate-down migrate-create sqlc templ docker-up docker-down clean install-tools run-all test-player test-hls

help:
	@echo "Available commands:"
	@echo "  make dev           - Run API server with hot reload (Air)"
	@echo "  make worker        - Run worker process"
	@echo "  make build         - Build production binaries"
	@echo "  make build-worker  - Build worker binary"
	@echo "  make run-all       - Start Docker, migrate, API server & worker"
	@echo "  make test          - Run tests with coverage"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback database migrations"
	@echo "  make migrate-create NAME=xxx - Create new migration"
	@echo "  make sqlc          - Generate SQLC code"
	@echo "  make templ         - Generate Templ templates"
	@echo "  make docker-up     - Start Docker services"
	@echo "  make docker-down   - Stop Docker services"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install-tools - Install required tools"
	@echo "  make test-player VIDEO_ID=xxx - Open video player in browser"
	@echo "  make test-hls VIDEO_ID=xxx - Test HLS playback with ffplay"

dev:
	air

worker:
	@go run cmd/worker/main.go

build:
	@echo "Building API server..."
	@go build -o bin/api cmd/api/main.go
	@echo "Building worker..."
	@go build -o bin/worker cmd/worker/main.go
	@echo "Build complete!"

build-worker:
	@echo "Building worker..."
	@go build -o bin/worker cmd/worker/main.go
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

migrate-up:
	@migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/video_streaming?sslmode=disable" up

migrate-down:
	@migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/video_streaming?sslmode=disable" down

migrate-create:
	@migrate create -ext sql -dir migrations -seq $(NAME)

sqlc:
	@sqlc generate
	@echo "SQLC code generated!"

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

clean:
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts!"

install-tools:
	@echo "Installing required tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/a-h/templ/cmd/templ@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed successfully!"

run:
	@go run cmd/api/main.go

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
