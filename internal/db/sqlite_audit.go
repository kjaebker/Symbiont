package db

import (
	"context"
	"fmt"
)

// --- Audit Events ---

// InsertAuditEvent inserts a row into the events table.
func (s *SQLiteDB) InsertAuditEvent(ctx context.Context, e AuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (ts, kind, payload_json, correlation_id) VALUES (?, ?, ?, ?)`,
		e.TS, e.Kind, e.PayloadJSON, e.CorrelationID,
	)
	if err != nil {
		return fmt.Errorf("inserting audit event kind=%s: %w", e.Kind, err)
	}
	return nil
}

// ListAuditEvents returns rows from the events table filtered by AuditFilter,
// ordered by ts DESC. Limit defaults to 100 and is capped at 500.
func (s *SQLiteDB) ListAuditEvents(ctx context.Context, f AuditFilter) ([]AuditEvent, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	query := `SELECT id, ts, kind, payload_json, correlation_id FROM events WHERE 1=1`
	var args []any

	if f.Kind != "" {
		query += ` AND kind = ?`
		args = append(args, f.Kind)
	}
	if f.Since != nil {
		query += ` AND ts >= ?`
		args = append(args, f.Since)
	}
	if f.CorrelationID != "" {
		query += ` AND correlation_id = ?`
		args = append(args, f.CorrelationID)
	}
	if f.InitiatedBy != "" {
		query += ` AND json_extract(payload_json, '$.data.initiated_by') = ?`
		args = append(args, f.InitiatedBy)
	}
	query += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}
	defer rows.Close()

	var out []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.TS, &e.Kind, &e.PayloadJSON, &e.CorrelationID); err != nil {
			return nil, fmt.Errorf("scanning audit event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
