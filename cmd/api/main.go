// Command api serves the HTTP API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nuu-maan/video-streaming-service/internal/app"
	"github.com/Nuu-maan/video-streaming-service/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Cancelled on the first SIGINT/SIGTERM, which is what tells App.Run to
	// begin a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer application.Close()

	return application.Run(ctx)
}
