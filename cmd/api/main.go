package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/api"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/poller"
)

func main() {
	// Set up structured JSON logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("SYMBIONT_LOG_LEVEL")),
	})).With("service", "api")
	slog.SetDefault(logger)

	// Load configuration.
	cfg := config.Load()

	// Open DuckDB read-write (single process owns the file).
	duckDB, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("failed to open duckdb", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}
	defer duckDB.Close()

	// Open SQLite read-write.
	sqliteDB, err := db.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		logger.Error("failed to open sqlite", "err", err, "path", cfg.SQLitePath)
		os.Exit(1)
	}
	defer sqliteDB.Close()

	// Bootstrap default token on first run.
	ctx := context.Background()
	token, created, err := sqliteDB.EnsureDefaultToken(ctx)
	if err != nil {
		logger.Error("failed to bootstrap token", "err", err)
		os.Exit(1)
	}
	if created {
		fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
		fmt.Printf("║  Symbiont API token (save this — shown once):                       ║\n")
		fmt.Printf("║  %s  ║\n", token)
		fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	}

	// Create Apex client.
	apexClient := apex.NewClient(cfg.ApexURL, cfg.ApexUser, cfg.ApexPass)

	// Set up signal-based context cancellation for graceful shutdown.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start the poller as a background goroutine. It shares the same DuckDB
	// connection as the API server — no file lock contention.
	pollerLogger := logger.With("component", "poller")
	p := poller.New(apexClient, duckDB, cfg.PollInterval, pollerLogger)
	go p.Run(sigCtx)

	// Create and run API server — blocks until context is cancelled.
	server := api.New(cfg, duckDB, sqliteDB, apexClient, logger)
	if err := server.Run(sigCtx); err != nil {
		logger.Error("api server error", "err", err)
		os.Exit(1)
	}

	logger.Info("api server shut down cleanly")
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
