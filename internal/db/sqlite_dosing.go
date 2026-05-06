package db

import (
	"context"
	"database/sql"
	"fmt"
)

// --- Dosing Products ---

// ListDosingProducts returns all products ordered by brand, name.
func (s *SQLiteDB) ListDosingProducts(ctx context.Context) ([]DosingProduct, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, brand, name, type, unit, notes, created_at
		 FROM dosing_products ORDER BY brand, name`)
	if err != nil {
		return nil, fmt.Errorf("listing dosing products: %w", err)
	}
	defer rows.Close()
	var out []DosingProduct
	for rows.Next() {
		var p DosingProduct
		if err := rows.Scan(&p.ID, &p.Brand, &p.Name, &p.Type, &p.Unit, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning dosing product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetDosingProduct returns a single product by ID.
func (s *SQLiteDB) GetDosingProduct(ctx context.Context, id int64) (*DosingProduct, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, brand, name, type, unit, notes, created_at
		 FROM dosing_products WHERE id = ?`, id)
	var p DosingProduct
	if err := row.Scan(&p.ID, &p.Brand, &p.Name, &p.Type, &p.Unit, &p.Notes, &p.CreatedAt); err != nil {
		return nil, fmt.Errorf("getting dosing product %d: %w", id, err)
	}
	return &p, nil
}

// InsertDosingProduct inserts a new product and returns its ID.
func (s *SQLiteDB) InsertDosingProduct(ctx context.Context, p DosingProduct) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO dosing_products (brand, name, type, unit, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		p.Brand, p.Name, p.Type, p.Unit, p.Notes)
	if err != nil {
		return 0, fmt.Errorf("inserting dosing product: %w", err)
	}
	return res.LastInsertId()
}

// UpdateDosingProduct updates a product's fields.
func (s *SQLiteDB) UpdateDosingProduct(ctx context.Context, p DosingProduct) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE dosing_products SET brand=?, name=?, type=?, unit=?, notes=? WHERE id=?`,
		p.Brand, p.Name, p.Type, p.Unit, p.Notes, p.ID)
	if err != nil {
		return fmt.Errorf("updating dosing product %d: %w", p.ID, err)
	}
	return nil
}

// DeleteDosingProduct deletes a product (cascades to schedules and logs).
func (s *SQLiteDB) DeleteDosingProduct(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dosing_products WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting dosing product %d: %w", id, err)
	}
	return nil
}

// SeedDosingProducts inserts a set of products if the table is empty.
func (s *SQLiteDB) SeedDosingProducts(ctx context.Context, products []DosingProduct) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dosing_products`).Scan(&count); err != nil {
		return fmt.Errorf("checking dosing products count: %w", err)
	}
	if count > 0 {
		return nil
	}
	for _, p := range products {
		if _, err := s.InsertDosingProduct(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// --- Dosing Schedules ---

// ListDosingSchedules returns all schedules joined with product info.
func (s *SQLiteDB) ListDosingSchedules(ctx context.Context) ([]DosingSchedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ds.id, ds.product_id, ds.amount, ds.frequency, ds.interval_days,
		        ds.day_of_week, ds.enabled, ds.last_completed_at, ds.next_due_at,
		        ds.follow_up_parameter_id, ds.follow_up_days, ds.notes, ds.created_at,
		        dp.brand, dp.name, dp.unit,
		        mp.name
		 FROM dosing_schedules ds
		 JOIN dosing_products dp ON dp.id = ds.product_id
		 LEFT JOIN measurement_parameters mp ON mp.id = ds.follow_up_parameter_id
		 ORDER BY dp.brand, dp.name, ds.id`)
	if err != nil {
		return nil, fmt.Errorf("listing dosing schedules: %w", err)
	}
	defer rows.Close()
	return scanDosingSchedules(rows)
}

// GetDosingSchedule returns a single schedule by ID.
func (s *SQLiteDB) GetDosingSchedule(ctx context.Context, id int64) (*DosingSchedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ds.id, ds.product_id, ds.amount, ds.frequency, ds.interval_days,
		        ds.day_of_week, ds.enabled, ds.last_completed_at, ds.next_due_at,
		        ds.follow_up_parameter_id, ds.follow_up_days, ds.notes, ds.created_at,
		        dp.brand, dp.name, dp.unit,
		        mp.name
		 FROM dosing_schedules ds
		 JOIN dosing_products dp ON dp.id = ds.product_id
		 LEFT JOIN measurement_parameters mp ON mp.id = ds.follow_up_parameter_id
		 WHERE ds.id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("getting dosing schedule %d: %w", id, err)
	}
	defer rows.Close()
	schedules, err := scanDosingSchedules(rows)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("dosing schedule %d: not found", id)
	}
	return &schedules[0], nil
}

func scanDosingSchedules(rows *sql.Rows) ([]DosingSchedule, error) {
	var out []DosingSchedule
	for rows.Next() {
		var d DosingSchedule
		if err := rows.Scan(
			&d.ID, &d.ProductID, &d.Amount, &d.Frequency, &d.IntervalDays,
			&d.DayOfWeek, &d.Enabled, &d.LastCompletedAt, &d.NextDueAt,
			&d.FollowUpParameterID, &d.FollowUpDays, &d.Notes, &d.CreatedAt,
			&d.ProductBrand, &d.ProductName, &d.ProductUnit,
			&d.FollowUpParameter,
		); err != nil {
			return nil, fmt.Errorf("scanning dosing schedule: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// InsertDosingSchedule inserts a new schedule and returns its ID.
func (s *SQLiteDB) InsertDosingSchedule(ctx context.Context, d DosingSchedule) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO dosing_schedules
		 (product_id, amount, frequency, interval_days, day_of_week,
		  enabled, follow_up_parameter_id, follow_up_days, notes, next_due_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ProductID, d.Amount, d.Frequency, d.IntervalDays, d.DayOfWeek,
		d.Enabled, d.FollowUpParameterID, d.FollowUpDays, d.Notes,
		computeNextDue(nil, d.Frequency, d.IntervalDays, d.DayOfWeek),
	)
	if err != nil {
		return 0, fmt.Errorf("inserting dosing schedule: %w", err)
	}
	return res.LastInsertId()
}

// UpdateDosingSchedule updates a schedule's configuration fields.
func (s *SQLiteDB) UpdateDosingSchedule(ctx context.Context, d DosingSchedule) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE dosing_schedules SET
		    product_id=?, amount=?, frequency=?, interval_days=?, day_of_week=?,
		    enabled=?, follow_up_parameter_id=?, follow_up_days=?, notes=?
		 WHERE id=?`,
		d.ProductID, d.Amount, d.Frequency, d.IntervalDays, d.DayOfWeek,
		d.Enabled, d.FollowUpParameterID, d.FollowUpDays, d.Notes, d.ID)
	if err != nil {
		return fmt.Errorf("updating dosing schedule %d: %w", d.ID, err)
	}
	return nil
}

// DeleteDosingSchedule deletes a schedule (logs' schedule_id becomes NULL via ON DELETE SET NULL).
func (s *SQLiteDB) DeleteDosingSchedule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dosing_schedules WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting dosing schedule %d: %w", id, err)
	}
	return nil
}

// --- Dosing Logs ---

// InsertDosingLog records a dose and advances the schedule's next_due_at.
// Returns the new log ID and the completed DosingLog (joined with product info).
func (s *SQLiteDB) InsertDosingLog(ctx context.Context, log DosingLog) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning dose log transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx,
		`INSERT INTO dosing_logs (schedule_id, product_id, amount, dosed_at, notes, source)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		log.ScheduleID, log.ProductID, log.Amount, log.DosedAt, log.Notes, log.Source)
	if err != nil {
		return 0, fmt.Errorf("inserting dosing log: %w", err)
	}
	logID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting dosing log id: %w", err)
	}

	// Advance the schedule's next_due_at if this was a scheduled dose.
	if log.ScheduleID != nil {
		var freq string
		var intervalDays *float64
		var dayOfWeek *int
		row := tx.QueryRowContext(ctx,
			`SELECT frequency, interval_days, day_of_week FROM dosing_schedules WHERE id=?`,
			*log.ScheduleID)
		if err := row.Scan(&freq, &intervalDays, &dayOfWeek); err != nil {
			return 0, fmt.Errorf("loading schedule for next_due_at: %w", err)
		}
		nextDue := computeNextDue(&log.DosedAt, freq, intervalDays, dayOfWeek)
		_, err = tx.ExecContext(ctx,
			`UPDATE dosing_schedules SET last_completed_at=?, next_due_at=? WHERE id=?`,
			log.DosedAt, nextDue, *log.ScheduleID)
		if err != nil {
			return 0, fmt.Errorf("updating schedule next_due_at: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing dose log: %w", err)
	}
	return logID, nil
}

// ListDosingLogs returns dose log entries joined with product info.
func (s *SQLiteDB) ListDosingLogs(ctx context.Context, f DosingLogFilter) ([]DosingLog, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	query := `SELECT dl.id, dl.schedule_id, dl.product_id, dl.amount, dl.dosed_at,
	                 dl.notes, dl.source, dl.created_at,
	                 dp.brand, dp.name, dp.unit
	          FROM dosing_logs dl
	          JOIN dosing_products dp ON dp.id = dl.product_id
	          WHERE 1=1`
	var args []any

	if f.ProductID != nil {
		query += ` AND dl.product_id = ?`
		args = append(args, *f.ProductID)
	}
	if f.ScheduleID != nil {
		query += ` AND dl.schedule_id = ?`
		args = append(args, *f.ScheduleID)
	}
	if f.From != nil {
		query += ` AND dl.dosed_at >= ?`
		args = append(args, *f.From)
	}
	if f.To != nil {
		query += ` AND dl.dosed_at <= ?`
		args = append(args, *f.To)
	}
	query += ` ORDER BY dl.dosed_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing dosing logs: %w", err)
	}
	defer rows.Close()

	var out []DosingLog
	for rows.Next() {
		var l DosingLog
		if err := rows.Scan(
			&l.ID, &l.ScheduleID, &l.ProductID, &l.Amount, &l.DosedAt,
			&l.Notes, &l.Source, &l.CreatedAt,
			&l.ProductBrand, &l.ProductName, &l.ProductUnit,
		); err != nil {
			return nil, fmt.Errorf("scanning dosing log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
