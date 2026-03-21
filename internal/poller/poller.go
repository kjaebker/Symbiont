package poller

import (
	"context"
	"log/slog"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
)

// Poller periodically fetches status from the Apex controller and writes
// the data to DuckDB.
type Poller struct {
	apex     apex.Client
	db       *db.DuckDB
	interval time.Duration
	logger   *slog.Logger
}

// New creates a new Poller.
func New(apexClient apex.Client, duckDB *db.DuckDB, interval time.Duration, logger *slog.Logger) *Poller {
	return &Poller{
		apex:     apexClient,
		db:       duckDB,
		interval: interval,
		logger:   logger,
	}
}

// Run starts the polling loop. It polls immediately, then on each tick.
// It blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	p.logger.Info("poller starting", "interval", p.interval.String())

	// Poll immediately on startup.
	p.poll(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poller stopping")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll executes a single poll cycle: fetch status from Apex, write to DuckDB.
func (p *Poller) poll(ctx context.Context) {
	start := time.Now()

	status, err := p.apex.Status(ctx)
	if err != nil {
		p.logger.Error("apex status fetch failed", "err", err)
		return
	}

	ts := time.Now()
	if err := p.db.WritePollCycle(ctx, ts, status); err != nil {
		p.logger.Error("duckdb write failed", "err", err)
		return
	}

	elapsed := time.Since(start)
	p.logger.Info("poll cycle complete",
		"duration_ms", elapsed.Milliseconds(),
		"probes", len(status.Inputs),
		"outlets", len(status.Outputs),
	)
}
