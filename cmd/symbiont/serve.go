package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kjaebker/symbiont/internal/alerts"
	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/api"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/notify"
	"github.com/kjaebker/symbiont/internal/poller"
	"github.com/spf13/cobra"
)

func newServeCmd(frontendFS fs.FS) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the Symbiont server (API + poller)",
		Long: `Starts the API server and poller in a single process.

Configure via environment variables or a .env file in the working directory.
Copy .env.example to .env and fill in your Apex credentials to get started.

The dashboard will be available at http://localhost:8420 (or SYMBIONT_API_PORT).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(frontendFS)
		},
	}
}

func runServe(frontendFS fs.FS) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv("SYMBIONT_LOG_LEVEL")),
	})).With("service", "symbiont")
	slog.SetDefault(logger)

	cfg := config.Load()

	duckDB, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening duckdb: %w", err)
	}
	defer duckDB.Close()

	sqliteDB, err := db.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("opening sqlite: %w", err)
	}
	defer sqliteDB.Close()

	ctx := context.Background()
	token, created, err := sqliteDB.EnsureDefaultToken(ctx)
	if err != nil {
		return fmt.Errorf("bootstrapping token: %w", err)
	}
	if created {
		fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
		fmt.Printf("║  Symbiont API token (save this — shown once):                       ║\n")
		fmt.Printf("║  %s  ║\n", token)
		fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	}

	apexClient := apex.NewClient(cfg.ApexURL, cfg.ApexUser, cfg.ApexPass)

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	pollerLogger := logger.With("component", "poller")
	p := poller.New(apexClient, duckDB, cfg.PollInterval, pollerLogger)
	if cfg.HeartbeatPath != "" {
		p.SetHeartbeatPath(cfg.HeartbeatPath)
	}
	go p.Run(sigCtx)

	server := api.New(cfg, duckDB, sqliteDB, apexClient, logger, frontendFS)

	targets, err := sqliteDB.ListEnabledNotificationTargets(ctx, "ntfy")
	if err != nil {
		logger.Warn("failed to load notification targets", "err", err)
	}
	var notifier notify.Notifier
	if len(targets) > 0 {
		var notifiers []notify.Notifier
		for _, t := range targets {
			notifiers = append(notifiers, notify.NewNtfy(t.Config))
		}
		notifier = notify.NewMulti(notifiers...)
		logger.Info("notification targets loaded", "count", len(targets))
	}

	alertLogger := logger.With("component", "alerts")
	alertEngine := alerts.New(sqliteDB, duckDB, notifier, server.Broadcaster(), alertLogger)
	go alertEngine.Start(sigCtx)

	logger.Info("symbiont starting", "port", cfg.APIPort, "url", "http://localhost:"+cfg.APIPort)
	if err := server.Run(sigCtx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	logger.Info("symbiont shut down cleanly")
	return nil
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
