package db

import (
	"context"
	"fmt"
	"time"
)

// CurrentProbeReadings returns the latest reading for each probe.
func (d *DuckDB) CurrentProbeReadings(ctx context.Context) ([]ProbeReading, error) {
	const query = `
		SELECT ts, probe_did, probe_type, probe_name, value
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY probe_did ORDER BY ts DESC) AS rn
			FROM probe_readings
		)
		WHERE rn = 1
		ORDER BY probe_name`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying current probe readings: %w", err)
	}
	defer rows.Close()

	var readings []ProbeReading
	for rows.Next() {
		var r ProbeReading
		if err := rows.Scan(&r.Timestamp, &r.DID, &r.Type, &r.Name, &r.Value); err != nil {
			return nil, fmt.Errorf("scanning probe reading: %w", err)
		}
		readings = append(readings, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating probe readings: %w", err)
	}
	return readings, nil
}

// ProbeHistory returns bucketed time-series data for a probe over a time range.
// The interval parameter is a DuckDB interval string (e.g., "5 minutes", "1 hour").
func (d *DuckDB) ProbeHistory(ctx context.Context, name string, from, to time.Time, interval string) ([]DataPoint, error) {
	query := fmt.Sprintf(`
		SELECT time_bucket(INTERVAL '%s', ts) AS bucket, AVG(value) AS avg_value
		FROM probe_readings
		WHERE probe_name = ? AND ts >= ? AND ts <= ?
		GROUP BY bucket
		ORDER BY bucket`, interval)

	rows, err := d.db.QueryContext(ctx, query, name, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying probe history for %s: %w", name, err)
	}
	defer rows.Close()

	var points []DataPoint
	for rows.Next() {
		var p DataPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, fmt.Errorf("scanning data point: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating probe history: %w", err)
	}
	return points, nil
}

// CurrentOutletStates returns the latest state for each outlet.
func (d *DuckDB) CurrentOutletStates(ctx context.Context) ([]OutletState, error) {
	const query = `
		SELECT ts, outlet_did, outlet_id, outlet_name, outlet_type, state, intensity
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY outlet_did ORDER BY ts DESC) AS rn
			FROM outlet_states
		)
		WHERE rn = 1
		ORDER BY outlet_name`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying current outlet states: %w", err)
	}
	defer rows.Close()

	var states []OutletState
	for rows.Next() {
		var s OutletState
		if err := rows.Scan(&s.Timestamp, &s.DID, &s.ID, &s.Name, &s.Type, &s.State, &s.Intensity); err != nil {
			return nil, fmt.Errorf("scanning outlet state: %w", err)
		}
		states = append(states, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating outlet states: %w", err)
	}
	return states, nil
}

// ControllerMeta returns the most recent controller metadata snapshot.
func (d *DuckDB) ControllerMeta(ctx context.Context) (*ControllerMeta, error) {
	const query = `
		SELECT ts, serial, hostname, software, hardware, controller_type, timezone
		FROM controller_meta
		ORDER BY ts DESC
		LIMIT 1`

	var m ControllerMeta
	err := d.db.QueryRowContext(ctx, query).Scan(
		&m.Timestamp, &m.Serial, &m.Hostname, &m.Software,
		&m.Hardware, &m.Type, &m.Timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("querying controller meta: %w", err)
	}
	return &m, nil
}

// LastPollTime returns the timestamp of the most recent probe reading.
func (d *DuckDB) LastPollTime(ctx context.Context) (time.Time, error) {
	const query = `SELECT MAX(ts) FROM probe_readings`

	var ts time.Time
	err := d.db.QueryRowContext(ctx, query).Scan(&ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("querying last poll time: %w", err)
	}
	return ts, nil
}
