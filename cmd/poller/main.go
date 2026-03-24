package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/poller"
)

func main() {
	// Set up structured JSON logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("SYMBIONT_LOG_LEVEL")),
	})).With("service", "poller")
	slog.SetDefault(logger)

	// Load configuration.
	cfg := config.Load()

	// Open DuckDB for writing.
	duckDB, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("failed to open duckdb", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}
	defer duckDB.Close()

	// Create Apex client.
	apexClient := apex.NewClient(cfg.ApexURL, cfg.ApexUser, cfg.ApexPass)

	// Set up signal-based context cancellation for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Create and run poller — blocks until context is cancelled.
	p := poller.New(apexClient, duckDB, cfg.PollInterval, logger)
	if cfg.HeartbeatPath != "" {
		p.SetHeartbeatPath(cfg.HeartbeatPath)
	}
	p.Run(ctx)

	logger.Info("poller shut down cleanly")
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
