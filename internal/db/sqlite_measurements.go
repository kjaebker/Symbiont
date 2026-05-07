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
