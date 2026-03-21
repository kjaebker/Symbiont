package db

import (
	"context"
	"fmt"
)

// CleanupResult holds the number of rows deleted from each table.
type CleanupResult struct {
	ProbeReadings  int64
	OutletStates   int64
	PowerEvents    int64
	ControllerMeta int64
}

// DeleteOldRows deletes rows older than retentionDays from all four DuckDB tables.
// Not wired to a timer yet — that happens in Phase 6.
func (d *DuckDB) DeleteOldRows(ctx context.Context, retentionDays int) (*CleanupResult, error) {
	tables := []struct {
		name string
		dest *int64
	}{
		{"probe_readings", nil},
		{"outlet_states", nil},
		{"power_events", nil},
		{"controller_meta", nil},
	}

	result := &CleanupResult{}
	tables[0].dest = &result.ProbeReadings
	tables[1].dest = &result.OutletStates
	tables[2].dest = &result.PowerEvents
	tables[3].dest = &result.ControllerMeta

	for _, t := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE ts < CURRENT_TIMESTAMP::TIMESTAMP - INTERVAL '%d days'", t.name, retentionDays)
		res, err := d.db.ExecContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("deleting old rows from %s: %w", t.name, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("getting rows affected for %s: %w", t.name, err)
		}
		*t.dest = n
	}

	return result, nil
}
