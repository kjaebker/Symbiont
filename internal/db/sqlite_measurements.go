package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// --- Measurement Parameters ---

// ListMeasurementParameters returns all parameters ordered by sort_order then name.
func (s *SQLiteDB) ListMeasurementParameters(ctx context.Context) ([]MeasurementParameter, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, canonical_unit, sort_order FROM measurement_parameters ORDER BY sort_order, name",
	)
	if err != nil {
		return nil, fmt.Errorf("listing measurement parameters: %w", err)
	}
	defer rows.Close()

	var params []MeasurementParameter
	for rows.Next() {
		var p MeasurementParameter
		if err := rows.Scan(&p.ID, &p.Name, &p.CanonicalUnit, &p.SortOrder); err != nil {
			return nil, fmt.Errorf("scanning measurement parameter: %w", err)
		}
		params = append(params, p)
	}
	return params, rows.Err()
}

// GetMeasurementParameterByName returns the parameter with the given name, or sql.ErrNoRows.
func (s *SQLiteDB) GetMeasurementParameterByName(ctx context.Context, name string) (*MeasurementParameter, error) {
	var p MeasurementParameter
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, canonical_unit, sort_order FROM measurement_parameters WHERE name = ?",
		name,
	).Scan(&p.ID, &p.Name, &p.CanonicalUnit, &p.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting measurement parameter %q: %w", name, err)
	}
	return &p, nil
}

// InsertMeasurementParameter adds a custom parameter and returns its ID.
func (s *SQLiteDB) InsertMeasurementParameter(ctx context.Context, name, unit string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO measurement_parameters (name, canonical_unit) VALUES (?, ?)",
		name, unit,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting measurement parameter %q: %w", name, err)
	}
	return res.LastInsertId()
}

// --- Measurements ---

// ListMeasurements returns measurements joined with their parameter name and unit.
func (s *SQLiteDB) ListMeasurements(ctx context.Context, f MeasurementFilter) ([]Measurement, error) {
	const base = `
		SELECT m.id, m.measured_at, m.parameter_id, m.value, m.notes, m.source,
		       m.test_kit_ref, m.raw_value, m.created_at, p.name, p.canonical_unit
		FROM measurements m
		JOIN measurement_parameters p ON p.id = m.parameter_id`

	var conditions []string
	var args []any

	if f.ParameterID != nil {
		conditions = append(conditions, "m.parameter_id = ?")
		args = append(args, *f.ParameterID)
	}
	if f.From != nil {
		conditions = append(conditions, "m.measured_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conditions = append(conditions, "m.measured_at <= ?")
		args = append(args, *f.To)
	}

	query := base
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	query += " ORDER BY m.measured_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing measurements: %w", err)
	}
	defer rows.Close()

	var measurements []Measurement
	for rows.Next() {
		var m Measurement
		if err := rows.Scan(
			&m.ID, &m.MeasuredAt, &m.ParameterID, &m.Value, &m.Notes, &m.Source,
			&m.TestKitRef, &m.RawValue, &m.CreatedAt, &m.Parameter, &m.CanonicalUnit,
		); err != nil {
			return nil, fmt.Errorf("scanning measurement: %w", err)
		}
		measurements = append(measurements, m)
	}
	return measurements, rows.Err()
}

// GetMeasurement returns a single measurement by ID, or nil if not found.
func (s *SQLiteDB) GetMeasurement(ctx context.Context, id int64) (*Measurement, error) {
	var m Measurement
	err := s.db.QueryRowContext(ctx, `
		SELECT m.id, m.measured_at, m.parameter_id, m.value, m.notes, m.source,
		       m.test_kit_ref, m.raw_value, m.created_at, p.name, p.canonical_unit
		FROM measurements m
		JOIN measurement_parameters p ON p.id = m.parameter_id
		WHERE m.id = ?`, id,
	).Scan(
		&m.ID, &m.MeasuredAt, &m.ParameterID, &m.Value, &m.Notes, &m.Source,
		&m.TestKitRef, &m.RawValue, &m.CreatedAt, &m.Parameter, &m.CanonicalUnit,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting measurement %d: %w", id, err)
	}
	return &m, nil
}

// InsertMeasurement inserts a new measurement and returns its ID.
func (s *SQLiteDB) InsertMeasurement(ctx context.Context, m Measurement) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO measurements (measured_at, parameter_id, value, notes, source, test_kit_ref, raw_value)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.MeasuredAt, m.ParameterID, m.Value, m.Notes, m.Source, m.TestKitRef, m.RawValue,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting measurement: %w", err)
	}
	return res.LastInsertId()
}

// UpdateMeasurement updates a measurement's mutable fields. Returns an error if not found.
func (s *SQLiteDB) UpdateMeasurement(ctx context.Context, id int64, m Measurement) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE measurements
		SET measured_at = ?, parameter_id = ?, value = ?, notes = ?, test_kit_ref = ?, raw_value = ?
		WHERE id = ?`,
		m.MeasuredAt, m.ParameterID, m.Value, m.Notes, m.TestKitRef, m.RawValue, id,
	)
	if err != nil {
		return fmt.Errorf("updating measurement %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("measurement %d not found", id)
	}
	return nil
}

// DeleteMeasurement removes a measurement by ID. Returns an error if not found.
func (s *SQLiteDB) DeleteMeasurement(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM measurements WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting measurement %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("measurement %d not found", id)
	}
	return nil
}

// MigrateDashboardLayout migrates old hidden/display_order data into dashboard_items.
// It is idempotent: if dashboard_items already has rows, it does nothing.
func (s *SQLiteDB) MigrateDashboardLayout(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dashboard_items").Scan(&count); err != nil {
		return fmt.Errorf("checking dashboard_items count: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning migration transaction: %w", err)
	}
	defer tx.Rollback()

	sortOrder := 1

	// Migrate probes. Fall back to simple query if legacy columns don't exist.
	probeRows, err := tx.QueryContext(ctx,
		"SELECT probe_name FROM probe_config WHERE hidden = 0 ORDER BY display_order, probe_name",
	)
	if err != nil {
		probeRows, err = tx.QueryContext(ctx, "SELECT probe_name FROM probe_config ORDER BY probe_name")
		if err != nil {
			return fmt.Errorf("reading probe configs for migration: %w", err)
		}
	}
	var probeNames []string
	for probeRows.Next() {
		var name string
		if err := probeRows.Scan(&name); err != nil {
			probeRows.Close()
			return fmt.Errorf("scanning probe name: %w", err)
		}
		probeNames = append(probeNames, name)
	}
	probeRows.Close()

	if len(probeNames) > 0 {
		// Add separator for probes.
		label := "Telemetry"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting probe separator: %w", err)
		}
		sortOrder++

		for _, name := range probeNames {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('probe', ?, NULL, ?)",
				name, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting probe dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	// Migrate devices. The hidden/display_order columns may not exist on
	// all production databases, so fall back to a simple query.
	deviceRows, err := tx.QueryContext(ctx,
		"SELECT id FROM devices WHERE hidden = 0 ORDER BY display_order, name",
	)
	if err != nil {
		// Columns don't exist — just select all devices.
		deviceRows, err = tx.QueryContext(ctx, "SELECT id FROM devices ORDER BY name")
		if err != nil {
			return fmt.Errorf("reading devices for migration: %w", err)
		}
	}
	var deviceIDs []string
	for deviceRows.Next() {
		var id int64
		if err := deviceRows.Scan(&id); err != nil {
			deviceRows.Close()
			return fmt.Errorf("scanning device id: %w", err)
		}
		deviceIDs = append(deviceIDs, fmt.Sprintf("%d", id))
	}
	deviceRows.Close()

	if len(deviceIDs) > 0 {
		label := "Equipment"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting device separator: %w", err)
		}
		sortOrder++

		for _, idStr := range deviceIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('device', ?, NULL, ?)",
				idStr, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting device dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	// Migrate outlets. Same fallback as devices.
	outletRows, err := tx.QueryContext(ctx,
		"SELECT outlet_id FROM outlet_config WHERE hidden = 0 ORDER BY display_order, outlet_id",
	)
	if err != nil {
		outletRows, err = tx.QueryContext(ctx, "SELECT outlet_id FROM outlet_config ORDER BY outlet_id")
		if err != nil {
			return fmt.Errorf("reading outlet configs for migration: %w", err)
		}
	}
	var outletIDs []string
	for outletRows.Next() {
		var id string
		if err := outletRows.Scan(&id); err != nil {
			outletRows.Close()
			return fmt.Errorf("scanning outlet id: %w", err)
		}
		outletIDs = append(outletIDs, id)
	}
	outletRows.Close()

	if len(outletIDs) > 0 {
		label := "Controls"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting outlet separator: %w", err)
		}
		sortOrder++

		for _, id := range outletIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('outlet', ?, NULL, ?)",
				id, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting outlet dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	return tx.Commit()
}
