// Package audit provides the event bus subscriber that writes every system
// event to the SQLite events table as a forensic audit log.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)

// Register subscribes the audit log writer to the event bus. Every system
// event published to the bus is serialized and persisted to the events table.
// Capacity 1024 keeps the audit log resilient to write bursts while isolating
// other subscribers from any temporary SQLite latency.
func Register(bus *events.Bus, sqlite *db.SQLiteDB, logger *slog.Logger) {
	bus.Subscribe("audit_log", 1024, func(ctx context.Context, e events.SystemEvent) {
		// Poll cycle completions are already captured in DuckDB with full probe
		// data. Persisting them here would generate ~8,640 rows/day of noise.
		if _, ok := e.(events.EvtPollCycleCompleted); ok {
			return
		}

		payload, err := marshal(e)
		if err != nil {
			logger.Error("audit: failed to marshal event payload", "kind", e.Kind(), "err", err)
			return
		}

		evt := db.AuditEvent{
			TS:          e.OccurredAt(),
			Kind:        e.Kind(),
			PayloadJSON: string(payload),
		}
		if err := sqlite.InsertAuditEvent(ctx, evt); err != nil {
			logger.Error("audit: failed to insert event", "kind", e.Kind(), "err", err)
		}
	})
}

// marshal serializes a SystemEvent to JSON for storage. Each concrete event
// type is serialized with its exported fields.
func marshal(e events.SystemEvent) ([]byte, error) {
	// Wrap with kind and ts for self-describing records.
	wrapper := struct {
		Kind string      `json:"kind"`
		TS   time.Time   `json:"ts"`
		Data interface{} `json:"data"`
	}{
		Kind: e.Kind(),
		TS:   e.OccurredAt(),
		Data: e,
	}
	return json.Marshal(wrapper)
}
