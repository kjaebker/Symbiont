package db

import (
	"context"
	"fmt"
	"io"
	"time"
)

// ExportProbeCSV streams probe readings for a probe name and time range as CSV to the writer.
func (d *DuckDB) ExportProbeCSV(ctx context.Context, w io.Writer, name string, from, to time.Time) error {
	const query = `SELECT ts, value FROM probe_readings WHERE probe_name = ? AND ts >= ? AND ts <= ? ORDER BY ts`

	rows, err := d.db.QueryContext(ctx, query, name, from, to)
	if err != nil {
		return fmt.Errorf("querying probe readings for export: %w", err)
	}
	defer rows.Close()

	// Write CSV header.
	if _, err := fmt.Fprint(w, "timestamp,value\n"); err != nil {
		return fmt.Errorf("writing csv header: %w", err)
	}

	for rows.Next() {
		var ts time.Time
		var value float64
		if err := rows.Scan(&ts, &value); err != nil {
			return fmt.Errorf("scanning export row: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s,%.4f\n", ts.Format(time.RFC3339), value); err != nil {
			return fmt.Errorf("writing csv row: %w", err)
		}
	}

	return rows.Err()
}

// ListProbeNames returns all distinct probe names in the database.
func (d *DuckDB) ListProbeNames(ctx context.Context) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT DISTINCT probe_name FROM probe_readings ORDER BY probe_name")
	if err != nil {
		return nil, fmt.Errorf("listing probe names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning probe name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
