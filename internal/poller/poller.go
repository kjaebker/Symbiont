package poller

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
)

// Poller periodically fetches status from the Apex controller and writes
// the data to DuckDB.
type Poller struct {
	apex          apex.Client
	db            *db.DuckDB
	interval      time.Duration
	logger        *slog.Logger
	heartbeatPath string
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

// SetHeartbeatPath configures a file path where the poller writes a heartbeat
// after each successful poll cycle. The file contains the PID and timestamp.
func (p *Poller) SetHeartbeatPath(path string) {
	p.heartbeatPath = path
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

	p.writeHeartbeat()
}

func (p *Poller) writeHeartbeat() {
	if p.heartbeatPath == "" {
		return
	}
	content := fmt.Sprintf("%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(p.heartbeatPath, []byte(content), 0o644); err != nil {
		p.logger.Warn("failed to write heartbeat", "err", err, "path", p.heartbeatPath)
	}
}

// ReadHeartbeat reads a heartbeat file and returns the timestamp.
// Returns zero time if the file doesn't exist or can't be parsed.
func ReadHeartbeat(path string) time.Time {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	// Format: "<pid> <RFC3339>\n"
	parts := splitOnSpace(string(data))
	if len(parts) < 2 {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, parts[1])
	if err != nil {
		return time.Time{}
	}
	return t
}

func splitOnSpace(s string) []string {
	var parts []string
	start := -1
	for i, c := range s {
		if c == ' ' || c == '\n' || c == '\r' {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}
