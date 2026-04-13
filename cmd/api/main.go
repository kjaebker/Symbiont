package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kjaebker/symbiont/internal/alerts"
	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/api"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
	"github.com/kjaebker/symbiont/internal/journal"
	"github.com/kjaebker/symbiont/internal/kits"
	"github.com/kjaebker/symbiont/internal/notify"
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

	// Load kit catalog, validated against seeded measurement parameters.
	measurementParams, err := sqliteDB.ListMeasurementParameters(ctx)
	if err != nil {
		logger.Error("failed to load measurement parameters for kit validation", "err", err)
		os.Exit(1)
	}
	validParams := make(map[string]bool, len(measurementParams))
	for _, mp := range measurementParams {
		validParams[mp.Name] = true
	}
	catalog, err := kits.Load(validParams)
	if err != nil {
		logger.Error("failed to load kit catalog", "err", err)
		os.Exit(1)
	}
	logger.Info("kit catalog loaded", "kits", len(catalog.All()))

	// Set up signal-based context cancellation for graceful shutdown.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start the poller as a background goroutine. It shares the same DuckDB
	// connection as the API server — no file lock contention.
	pollerLogger := logger.With("component", "poller")
	p := poller.New(apexClient, duckDB, cfg.PollInterval, pollerLogger)
	if cfg.HeartbeatPath != "" {
		p.SetHeartbeatPath(cfg.HeartbeatPath)
	}
	go p.Run(sigCtx)

	// Load journal template catalog.
	journalCatalog, err := journal.Load()
	if err != nil {
		logger.Error("failed to load journal templates", "err", err)
		os.Exit(1)
	}
	logger.Info("journal templates loaded", "count", len(journalCatalog.All()))

	// Create event bus and wire the journal auto-log subscriber.
	bus := events.NewBus(logger.With("component", "events"))
	bus.Subscribe("journal_autolog", 256, func(ctx context.Context, e events.SystemEvent) {
		if entry, ok := journal.EntryFromEvent(e); ok {
			if _, err := sqliteDB.InsertJournalEntry(ctx, entry); err != nil {
				logger.Error("journal auto-log failed", "err", err, "kind", e.Kind())
			}
		}
	})

	// Start the event bus dispatchers.
	bus.Start(sigCtx)

	// Create API server (nil FS falls back to cfg.FrontendPath).
	server := api.New(cfg, duckDB, sqliteDB, apexClient, logger, nil, catalog, bus, journalCatalog)

	// Build notification system from enabled targets.
	var notifier notify.Notifier
	targets, err := sqliteDB.ListEnabledNotificationTargets(ctx, "ntfy")
	if err != nil {
		logger.Warn("failed to load notification targets", "err", err)
	}
	if len(targets) > 0 {
		var notifiers []notify.Notifier
		for _, t := range targets {
			notifiers = append(notifiers, notify.NewNtfy(t.Config))
		}
		notifier = notify.NewMulti(notifiers...)
		logger.Info("notification targets loaded", "count", len(targets))
	}

	// Start alert engine.
	alertLogger := logger.With("component", "alerts")
	alertEngine := alerts.New(sqliteDB, duckDB, notifier, server.Broadcaster(), alertLogger)
	go alertEngine.Start(sigCtx)

	// Run API server — blocks until context is cancelled.
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
