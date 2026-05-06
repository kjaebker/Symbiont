package db

import (
	"context"
	"fmt"
	"time"
)

// --- Outlet Programs ---

// ProgramRecord is a single Apex output program stored in SQLite.
type ProgramRecord struct {
	DID       string
	Name      string
	Type      string
	Icon      string
	Prog      string
	FetchedAt time.Time
	UpdatedAt time.Time
}

// UpsertOutletPrograms writes program records to outlet_programs, tracking
// content changes in outlet_program_history. Returns the number of programs
// whose prog content changed.
func (s *SQLiteDB) UpsertOutletPrograms(ctx context.Context, programs []ProgramRecord) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Load all current prog values in one query.
	rows, err := tx.QueryContext(ctx, `SELECT did, prog FROM outlet_programs`)
	if err != nil {
		return 0, fmt.Errorf("loading current programs: %w", err)
	}
	current := make(map[string]string)
	for rows.Next() {
		var did, prog string
		if err := rows.Scan(&did, &prog); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning current program: %w", err)
		}
		current[did] = prog
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating current programs: %w", err)
	}

	changed := 0
	now := time.Now().UTC()

	for _, p := range programs {
		prev, exists := current[p.DID]
		progChanged := !exists || prev != p.Prog

		if progChanged {
			changed++
		}

		// Use updated_at = now only when prog content changes.
		updatedAt := p.UpdatedAt
		if !progChanged {
			updatedAt = time.Time{} // placeholder; will be overridden by CASE below
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO outlet_programs (did, name, type, icon, prog, fetched_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(did) DO UPDATE SET
			   name       = excluded.name,
			   type       = excluded.type,
			   icon       = excluded.icon,
			   prog       = excluded.prog,
			   fetched_at = excluded.fetched_at,
			   updated_at = CASE WHEN outlet_programs.prog != excluded.prog
			                     THEN excluded.fetched_at
			                     ELSE outlet_programs.updated_at END`,
			p.DID, p.Name, p.Type, p.Icon, p.Prog, now, now,
		)
		if err != nil {
			return 0, fmt.Errorf("upserting program %s: %w", p.DID, err)
		}

		if progChanged {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO outlet_program_history (did, prog, changed_at) VALUES (?, ?, ?)`,
				p.DID, p.Prog, now,
			)
			if err != nil {
				return 0, fmt.Errorf("inserting program history for %s: %w", p.DID, err)
			}
		}

		_ = updatedAt // resolved via SQL CASE above
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing program upsert: %w", err)
	}
	return changed, nil
}

// ListOutletPrograms returns all stored outlet programs ordered by name.
func (s *SQLiteDB) ListOutletPrograms(ctx context.Context) ([]ProgramRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT did, name, type, icon, prog, fetched_at, updated_at
		 FROM outlet_programs
		 ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing outlet programs: %w", err)
	}
	defer rows.Close()

	var out []ProgramRecord
	for rows.Next() {
		var p ProgramRecord
		if err := rows.Scan(&p.DID, &p.Name, &p.Type, &p.Icon, &p.Prog,
			&p.FetchedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning outlet program: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// OutletProgramCount returns the number of stored outlet programs.
func (s *SQLiteDB) OutletProgramCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outlet_programs`).Scan(&n)
	return n, err
}

// computeNextDue computes the next_due_at timestamp based on frequency.
// from is the completion time; nil means now.
func computeNextDue(from *time.Time, frequency string, intervalDays *float64, dayOfWeek *int) *time.Time {
	if frequency == "as_needed" {
		return nil
	}
	base := time.Now()
	if from != nil {
		base = *from
	}

	var next time.Time
	switch frequency {
	case "daily":
		y, m, d := base.Date()
		next = time.Date(y, m, d+1, 0, 0, 0, 0, base.Location())
	case "twice_daily":
		next = base.Add(12 * time.Hour)
	case "every_n_days":
		days := 1.0
		if intervalDays != nil {
			days = *intervalDays
		}
		next = base.Add(time.Duration(days*24) * time.Hour)
	case "weekly":
		if dayOfWeek != nil {
			// Advance to the next occurrence of this weekday.
			target := time.Weekday(*dayOfWeek)
			next = base.AddDate(0, 0, 1)
			for next.Weekday() != target {
				next = next.AddDate(0, 0, 1)
			}
		} else {
			next = base.AddDate(0, 0, 7)
		}
	case "monthly":
		next = base.AddDate(0, 1, 0)
	default:
		next = base.Add(24 * time.Hour)
	}
	return &next
}
