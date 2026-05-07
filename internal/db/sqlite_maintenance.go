package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// --- Maintenance Tasks ---

// ListMaintenanceTasks returns all tasks ordered by name.
func (s *SQLiteDB) ListMaintenanceTasks(ctx context.Context) ([]MaintenanceTask, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, frequency, interval_days, day_of_week,
		        enabled, last_completed_at, next_due_at, created_at
		 FROM maintenance_tasks ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing maintenance tasks: %w", err)
	}
	defer rows.Close()
	return scanMaintenanceTasks(rows)
}

// GetMaintenanceTask returns a single task by ID.
func (s *SQLiteDB) GetMaintenanceTask(ctx context.Context, id int64) (*MaintenanceTask, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, frequency, interval_days, day_of_week,
		        enabled, last_completed_at, next_due_at, created_at
		 FROM maintenance_tasks WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("getting maintenance task %d: %w", id, err)
	}
	defer rows.Close()
	tasks, err := scanMaintenanceTasks(rows)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("maintenance task %d: not found", id)
	}
	return &tasks[0], nil
}

func scanMaintenanceTasks(rows *sql.Rows) ([]MaintenanceTask, error) {
	var out []MaintenanceTask
	for rows.Next() {
		var t MaintenanceTask
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Description, &t.Frequency, &t.IntervalDays,
			&t.DayOfWeek, &t.Enabled, &t.LastCompletedAt, &t.NextDueAt, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning maintenance task: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// InsertMaintenanceTask inserts a new task and returns its ID.
func (s *SQLiteDB) InsertMaintenanceTask(ctx context.Context, t MaintenanceTask) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO maintenance_tasks
		 (name, description, frequency, interval_days, day_of_week, enabled, next_due_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Description, t.Frequency, t.IntervalDays, t.DayOfWeek, t.Enabled,
		computeNextDue(nil, t.Frequency, t.IntervalDays, t.DayOfWeek),
	)
	if err != nil {
		return 0, fmt.Errorf("inserting maintenance task: %w", err)
	}
	return res.LastInsertId()
}

// UpdateMaintenanceTask updates a task's configuration fields.
func (s *SQLiteDB) UpdateMaintenanceTask(ctx context.Context, t MaintenanceTask) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE maintenance_tasks SET
		    name=?, description=?, frequency=?, interval_days=?,
		    day_of_week=?, enabled=?
		 WHERE id=?`,
		t.Name, t.Description, t.Frequency, t.IntervalDays, t.DayOfWeek, t.Enabled, t.ID)
	if err != nil {
		return fmt.Errorf("updating maintenance task %d: %w", t.ID, err)
	}
	return nil
}

// DeleteMaintenanceTask deletes a task and its logs.
func (s *SQLiteDB) DeleteMaintenanceTask(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM maintenance_tasks WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting maintenance task %d: %w", id, err)
	}
	return nil
}

// --- Maintenance Logs ---

// InsertMaintenanceLog records a task completion and advances next_due_at.
func (s *SQLiteDB) InsertMaintenanceLog(ctx context.Context, log MaintenanceLog) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning maintenance log transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx,
		`INSERT INTO maintenance_logs (task_id, completed_at, notes, source)
		 VALUES (?, ?, ?, ?)`,
		log.TaskID, log.CompletedAt, log.Notes, log.Source)
	if err != nil {
		return 0, fmt.Errorf("inserting maintenance log: %w", err)
	}
	logID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting maintenance log id: %w", err)
	}

	var freq string
	var intervalDays *float64
	var dayOfWeek *int
	row := tx.QueryRowContext(ctx,
		`SELECT frequency, interval_days, day_of_week FROM maintenance_tasks WHERE id=?`,
		log.TaskID)
	if err := row.Scan(&freq, &intervalDays, &dayOfWeek); err != nil {
		return 0, fmt.Errorf("loading task for next_due_at: %w", err)
	}
	nextDue := computeNextDue(&log.CompletedAt, freq, intervalDays, dayOfWeek)
	_, err = tx.ExecContext(ctx,
		`UPDATE maintenance_tasks SET last_completed_at=?, next_due_at=? WHERE id=?`,
		log.CompletedAt, nextDue, log.TaskID)
	if err != nil {
		return 0, fmt.Errorf("updating maintenance task next_due_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing maintenance log: %w", err)
	}
	return logID, nil
}

// ListMaintenanceLogs returns log entries for a task, ordered by completed_at DESC.
func (s *SQLiteDB) ListMaintenanceLogs(ctx context.Context, taskID int64, limit int) ([]MaintenanceLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT ml.id, ml.task_id, ml.completed_at, ml.notes, ml.source, ml.created_at,
		        mt.name
		 FROM maintenance_logs ml
		 JOIN maintenance_tasks mt ON mt.id = ml.task_id
		 WHERE ml.task_id = ?
		 ORDER BY ml.completed_at DESC LIMIT ?`,
		taskID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing maintenance logs: %w", err)
	}
	defer rows.Close()
	var out []MaintenanceLog
	for rows.Next() {
		var l MaintenanceLog
		if err := rows.Scan(
			&l.ID, &l.TaskID, &l.CompletedAt, &l.Notes, &l.Source, &l.CreatedAt,
			&l.TaskName,
		); err != nil {
			return nil, fmt.Errorf("scanning maintenance log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// --- Due Items ---

// ListDueItems returns all enabled dosing schedules and maintenance tasks that
// are due or overdue (next_due_at <= horizon). Horizon is typically now+24h.
func (s *SQLiteDB) ListDueItems(ctx context.Context, horizon time.Time) ([]DueItem, error) {
	now := time.Now()
	var out []DueItem

	// Due dosing schedules
	rows, err := s.db.QueryContext(ctx,
		`SELECT ds.id, ds.amount, ds.frequency, ds.next_due_at,
		        dp.brand, dp.name, dp.unit, dp.id
		 FROM dosing_schedules ds
		 JOIN dosing_products dp ON dp.id = ds.product_id
		 WHERE ds.enabled = 1
		   AND (ds.next_due_at IS NULL OR ds.next_due_at <= ?)
		   AND ds.frequency != 'as_needed'
		 ORDER BY ds.next_due_at ASC`, horizon)
	if err != nil {
		return nil, fmt.Errorf("listing due dosing schedules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id        int64
			amount    float64
			freq      string
			nextDue   *time.Time
			brand     string
			name      string
			unit      string
			productID int64
		)
		if err := rows.Scan(&id, &amount, &freq, &nextDue, &brand, &name, &unit, &productID); err != nil {
			return nil, fmt.Errorf("scanning due dosing schedule: %w", err)
		}
		out = append(out, DueItem{
			Kind:        "dose",
			ID:          id,
			Label:       brand + " " + name,
			Detail:      fmt.Sprintf("%.4g %s", amount, unit),
			NextDueAt:   nextDue,
			IsOverdue:   nextDue != nil && nextDue.Before(now),
			ProductID:   &productID,
			ProductUnit: &unit,
			Amount:      &amount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Due maintenance tasks — exclude daily tasks already completed today.
	rows2, err := s.db.QueryContext(ctx,
		`SELECT id, name, frequency, next_due_at
		 FROM maintenance_tasks
		 WHERE enabled = 1
		   AND (next_due_at IS NULL OR next_due_at <= ?)
		   AND frequency != 'as_needed'
		   AND NOT (frequency = 'daily'
		            AND last_completed_at IS NOT NULL
		            AND date(last_completed_at, 'localtime') = date('now', 'localtime'))
		 ORDER BY next_due_at ASC`, horizon)
	if err != nil {
		return nil, fmt.Errorf("listing due maintenance tasks: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var (
			id      int64
			name    string
			freq    string
			nextDue *time.Time
		)
		if err := rows2.Scan(&id, &name, &freq, &nextDue); err != nil {
			return nil, fmt.Errorf("scanning due maintenance task: %w", err)
		}
		out = append(out, DueItem{
			Kind:      "task",
			ID:        id,
			Label:     name,
			Detail:    freq,
			NextDueAt: nextDue,
			IsOverdue: nextDue != nil && nextDue.Before(now),
		})
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
