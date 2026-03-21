package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"

	_ "github.com/marcboeker/go-duckdb"
)

// DuckDB wraps a *sql.DB connection to a DuckDB database.
type DuckDB struct {
	db *sql.DB
}

// DB returns the underlying *sql.DB for use in read queries.
func (d *DuckDB) DB() *sql.DB {
	return d.db
}

// Open opens a read-write DuckDB connection at the given path and creates
// the schema if it does not already exist.
func Open(path string) (*DuckDB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb at %s: %w", path, err)
	}

	if err := CreateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating duckdb schema: %w", err)
	}

	if err := MigrateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating duckdb schema: %w", err)
	}

	return &DuckDB{db: db}, nil
}

// OpenReadOnly opens a read-only DuckDB connection at the given path.
func OpenReadOnly(path string) (*DuckDB, error) {
	dsn := path + "?access_mode=read_only"
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb read-only at %s: %w", path, err)
	}
	return &DuckDB{db: db}, nil
}

// Close closes the underlying database connection.
func (d *DuckDB) Close() error {
	return d.db.Close()
}

// WriteProbeReadings batch-inserts probe readings into DuckDB.
func (d *DuckDB) WriteProbeReadings(ctx context.Context, ts time.Time, inputs []apex.Input) error {
	return d.writeProbeReadingsTx(ctx, nil, ts, inputs)
}

func (d *DuckDB) writeProbeReadingsTx(ctx context.Context, tx *sql.Tx, ts time.Time, inputs []apex.Input) error {
	const query = `INSERT INTO probe_readings (ts, probe_did, probe_type, probe_name, value) VALUES (?, ?, ?, ?, ?)`

	var execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = d.db
	}

	for _, input := range inputs {
		if _, err := execer.ExecContext(ctx, query, ts, input.DID, input.Type, input.Name, input.Value); err != nil {
			return fmt.Errorf("inserting probe reading %s: %w", input.Name, err)
		}
	}
	return nil
}

// WriteOutletStates batch-inserts outlet states into DuckDB.
func (d *DuckDB) WriteOutletStates(ctx context.Context, ts time.Time, outputs []apex.Output) error {
	return d.writeOutletStatesTx(ctx, nil, ts, outputs)
}

func (d *DuckDB) writeOutletStatesTx(ctx context.Context, tx *sql.Tx, ts time.Time, outputs []apex.Output) error {
	const query = `INSERT INTO outlet_states (ts, outlet_did, outlet_id, outlet_name, outlet_type, state, intensity) VALUES (?, ?, ?, ?, ?, ?, ?)`

	var execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = d.db
	}

	for _, output := range outputs {
		state := output.State()
		intensity := 0
		if output.Intensity != nil {
			intensity = *output.Intensity
		}

		if _, err := execer.ExecContext(ctx, query, ts, output.DID, output.ID, output.Name, output.Type, state, intensity); err != nil {
			return fmt.Errorf("inserting outlet state %s: %w", output.Name, err)
		}
	}
	return nil
}

// WritePowerEvents inserts power events into DuckDB, deduplicating by event_ts.
// Only inserts an event if no row with the same event_ts already exists.
func (d *DuckDB) WritePowerEvents(ctx context.Context, ts time.Time, power apex.PowerInfo) error {
	return d.writePowerEventsTx(ctx, nil, ts, power)
}

func (d *DuckDB) writePowerEventsTx(ctx context.Context, tx *sql.Tx, ts time.Time, power apex.PowerInfo) error {
	const checkQuery = `SELECT COUNT(*) FROM power_events WHERE event_type = ? AND event_ts = ?`
	const insertQuery = `INSERT INTO power_events (ts, event_type, event_ts) VALUES (?, ?, ?)`

	type execQueryer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}

	var eq execQueryer
	if tx != nil {
		eq = tx
	} else {
		eq = d.db
	}

	insertIfNew := func(eventType string, epoch int64) error {
		if epoch == 0 {
			return nil
		}

		var count int
		if err := eq.QueryRowContext(ctx, checkQuery, eventType, epoch).Scan(&count); err != nil {
			return fmt.Errorf("checking existing %s event: %w", eventType, err)
		}
		if count > 0 {
			return nil
		}
		if _, err := eq.ExecContext(ctx, insertQuery, ts, eventType, epoch); err != nil {
			return fmt.Errorf("inserting power %s event: %w", eventType, err)
		}
		return nil
	}

	if err := insertIfNew("failed", power.Failed); err != nil {
		return err
	}
	if err := insertIfNew("restored", power.Restored); err != nil {
		return err
	}
	return nil
}

// WriteControllerMeta inserts a controller metadata snapshot into DuckDB.
func (d *DuckDB) WriteControllerMeta(ctx context.Context, ts time.Time, sys apex.SystemInfo) error {
	return d.writeControllerMetaTx(ctx, nil, ts, sys)
}

func (d *DuckDB) writeControllerMetaTx(ctx context.Context, tx *sql.Tx, ts time.Time, sys apex.SystemInfo) error {
	const query = `INSERT INTO controller_meta (ts, serial, hostname, software, hardware, controller_type, timezone) VALUES (?, ?, ?, ?, ?, ?, ?)`

	var execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = d.db
	}

	if _, err := execer.ExecContext(ctx, query, ts, sys.Serial, sys.Hostname, sys.Software, sys.Hardware, sys.Type, sys.Timezone); err != nil {
		return fmt.Errorf("inserting controller meta: %w", err)
	}
	return nil
}

// WritePollCycle writes all data from a single poll cycle in a single transaction.
func (d *DuckDB) WritePollCycle(ctx context.Context, ts time.Time, status *apex.StatusResponse) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning poll cycle transaction: %w", err)
	}
	defer tx.Rollback()

	if err := d.writeProbeReadingsTx(ctx, tx, ts, status.Inputs); err != nil {
		return fmt.Errorf("writing probe readings: %w", err)
	}

	if err := d.writeOutletStatesTx(ctx, tx, ts, status.Outputs); err != nil {
		return fmt.Errorf("writing outlet states: %w", err)
	}

	if err := d.writePowerEventsTx(ctx, tx, ts, status.Power); err != nil {
		return fmt.Errorf("writing power events: %w", err)
	}

	if err := d.writeControllerMetaTx(ctx, tx, ts, status.System); err != nil {
		return fmt.Errorf("writing controller meta: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing poll cycle transaction: %w", err)
	}
	return nil
}
