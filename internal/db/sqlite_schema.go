package db

import (
	"database/sql"
	"fmt"
	"sort"
)

// Migration is a versioned schema change. Migrations run inside a transaction;
// the runner records each successful application in `schema_versions`.
//
// Never edit a Migration after it has been merged. To revise a published
// schema, add a new Migration with a higher Version.
type Migration struct {
	Version int
	Name    string
	Up      func(*sql.Tx) error
}

// migrations is the ordered list of schema versions beyond the v1 baseline.
// Append new entries here; do not edit existing ones. Versions must be
// strictly increasing.
//
// The v1 baseline itself is created by createSchemaV1, not listed here.
var migrations = []Migration{
	{
		Version: 2,
		Name:    "alert_events partial index for active rows",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_alert_events_active
				ON alert_events(fired_at DESC) WHERE cleared_at IS NULL`)
			return err
		},
	},
	{
		Version: 3,
		Name:    "promote events.initiated_by to a real column",
		Up: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`ALTER TABLE events ADD COLUMN initiated_by TEXT`); err != nil {
				return fmt.Errorf("adding events.initiated_by column: %w", err)
			}
			// Backfill from JSON for any rows that already had it embedded.
			if _, err := tx.Exec(`UPDATE events
				SET initiated_by = json_extract(payload_json, '$.data.initiated_by')
				WHERE initiated_by IS NULL
				  AND json_extract(payload_json, '$.data.initiated_by') IS NOT NULL`); err != nil {
				return fmt.Errorf("backfilling events.initiated_by: %w", err)
			}
			if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_events_initiated_by
				ON events(initiated_by, ts DESC) WHERE initiated_by IS NOT NULL`); err != nil {
				return fmt.Errorf("creating idx_events_initiated_by: %w", err)
			}
			return nil
		},
	},
}

// CreateSQLiteSchema is the entrypoint preserved for callers. It ensures the
// schema_versions table exists, applies the v1 baseline if this database has
// never been initialised, then runs any unapplied versioned migrations.
func CreateSQLiteSchema(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("setting %s: %w", p, err)
		}
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_versions (
		version    INTEGER  PRIMARY KEY,
		name       TEXT     NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("creating schema_versions: %w", err)
	}

	applied, err := loadAppliedVersions(db)
	if err != nil {
		return err
	}

	if !applied[1] {
		if err := applyV1Baseline(db); err != nil {
			return fmt.Errorf("applying v1 baseline: %w", err)
		}
	}

	pending := make([]Migration, 0, len(migrations))
	for _, m := range migrations {
		if !applied[m.Version] {
			pending = append(pending, m)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	for _, m := range pending {
		if m.Version <= 1 {
			return fmt.Errorf("migration version %d (%s) must be > 1; v1 is the baseline", m.Version, m.Name)
		}
		if err := runMigration(db, m); err != nil {
			return err
		}
	}
	return nil
}

func loadAppliedVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_versions`)
	if err != nil {
		return nil, fmt.Errorf("reading schema_versions: %w", err)
	}
	defer rows.Close()
	applied := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scanning schema_versions row: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// applyV1Baseline creates every table and index that the schema declares as of
// v1. The CREATE TABLE / CREATE INDEX statements use IF NOT EXISTS so an
// existing database (predating schema_versions) keeps its tables — only the
// schema_versions row is inserted, recording that v1 has been applied.
func applyV1Baseline(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := createSchemaV1(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_versions(version, name) VALUES (1, ?)`,
		"initial schema",
	); err != nil {
		return fmt.Errorf("recording v1: %w", err)
	}
	return tx.Commit()
}

func runMigration(db *sql.DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration v%d: %w", m.Version, err)
	}
	defer tx.Rollback()
	if err := m.Up(tx); err != nil {
		return fmt.Errorf("migration v%d (%s): %w", m.Version, m.Name, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO schema_versions(version, name) VALUES (?, ?)`,
		m.Version, m.Name,
	); err != nil {
		return fmt.Errorf("recording v%d: %w", m.Version, err)
	}
	return tx.Commit()
}

// createSchemaV1 is the consolidated baseline schema: every column, every
// CHECK constraint, and every index that previously accumulated through
// ad-hoc ALTERs and table-recreation migrations is inlined here.
func createSchemaV1(tx *sql.Tx) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS auth_tokens (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			token       TEXT     NOT NULL UNIQUE,
			label       TEXT,
			scope       TEXT     NOT NULL DEFAULT 'admin',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used   DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			name          TEXT     NOT NULL,
			device_type   TEXT,
			description   TEXT,
			brand         TEXT,
			model         TEXT,
			notes         TEXT,
			image_path    TEXT,
			outlet_id     TEXT     UNIQUE,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS probe_config (
			probe_name      TEXT    PRIMARY KEY,
			display_name    TEXT,
			unit_override   TEXT,
			min_normal      REAL,
			max_normal      REAL,
			min_warning     REAL,
			max_warning     REAL,
			device_id       INTEGER REFERENCES devices(id) ON DELETE SET NULL,
			input_category  TEXT    NOT NULL DEFAULT 'probe',
			on_label        TEXT,
			off_label       TEXT,
			ok_value        REAL,
			is_binary       INTEGER NOT NULL DEFAULT 0,
			hidden          INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_config (
			outlet_id       TEXT PRIMARY KEY,
			display_name    TEXT,
			icon            TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS device_outlets (
			device_id   INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
			outlet_id   TEXT    NOT NULL,
			label       TEXT,
			color       TEXT,
			sort_order  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (device_id, outlet_id)
		)`,
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			probe_name      TEXT     NOT NULL,
			condition       TEXT     NOT NULL CHECK(condition IN ('above','below','outside_range')),
			threshold_low   REAL,
			threshold_high  REAL,
			severity        TEXT     NOT NULL CHECK(severity IN ('warning','critical')),
			cooldown_minutes INTEGER NOT NULL DEFAULT 30,
			enabled         INTEGER  NOT NULL DEFAULT 1,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_events (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			rule_id         INTEGER  NOT NULL REFERENCES alert_rules(id),
			fired_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cleared_at      DATETIME,
			peak_value      REAL,
			notified        INTEGER  NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS notification_targets (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			type        TEXT     NOT NULL,
			config      TEXT     NOT NULL,
			label       TEXT,
			enabled     INTEGER  NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS backup_jobs (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status      TEXT     NOT NULL CHECK(status IN ('success','failed')),
			path        TEXT,
			size_bytes  INTEGER,
			error       TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS dashboard_items (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			item_type    TEXT    NOT NULL CHECK(item_type IN ('probe','outlet','device','separator','feed_mode','measurement')),
			reference_id TEXT,
			label        TEXT,
			sort_order   INTEGER NOT NULL,
			display_mode TEXT    NOT NULL DEFAULT 'normal'
		)`,
		`CREATE TABLE IF NOT EXISTS measurement_parameters (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			name           TEXT    NOT NULL UNIQUE,
			canonical_unit TEXT    NOT NULL,
			sort_order     INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS measurements (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			measured_at   DATETIME NOT NULL,
			parameter_id  INTEGER  NOT NULL REFERENCES measurement_parameters(id),
			value         REAL     NOT NULL,
			notes         TEXT,
			source        TEXT     NOT NULL DEFAULT 'manual'
			              CHECK(source IN ('manual','import')),
			test_kit_ref  TEXT,
			raw_value     REAL,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS livestock (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     NOT NULL,
			species    TEXT,
			type       TEXT     NOT NULL CHECK(type IN ('fish','coral','invertebrate','other')),
			quantity   INTEGER  NOT NULL DEFAULT 1,
			status     TEXT     NOT NULL DEFAULT 'healthy' CHECK(status IN ('healthy','sick','quarantine','deceased')),
			date_added DATE,
			notes      TEXT,
			image_path TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS livestock_observations (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			livestock_id INTEGER  NOT NULL REFERENCES livestock(id) ON DELETE CASCADE,
			ts           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status       TEXT     CHECK(status IN ('healthy','sick','quarantine','deceased')),
			note         TEXT,
			image_path   TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS tank_profile (
			section         TEXT NOT NULL PRIMARY KEY CHECK(section IN ('display','sump')),
			display_name    TEXT,
			volume_gallons  REAL,
			length_in       REAL,
			width_in        REAL,
			height_in       REAL,
			tank_type       TEXT,
			manufacturer    TEXT,
			model           TEXT,
			setup_date      DATE,
			notes           TEXT,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS journal_entries (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			category    TEXT     NOT NULL CHECK(category IN ('observation','maintenance','event','milestone')),
			sentiment   TEXT     CHECK(sentiment IN ('good','neutral','bad','critical')),
			title       TEXT     NOT NULL,
			body        TEXT,
			source      TEXT     NOT NULL DEFAULT 'manual' CHECK(source IN ('manual','system','ai')),
			source_ref  TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS agent_settings (
			id                  INTEGER  PRIMARY KEY CHECK(id = 1),
			tone                TEXT     NOT NULL DEFAULT 'analytical'
			                    CHECK(tone IN ('analytical','casual','terse')),
			dosing_product_line TEXT     NOT NULL DEFAULT 'generic',
			net_volume_gallons  REAL,
			custom_guardrails   TEXT,
			enabled_skills      TEXT     NOT NULL DEFAULT '[]',
			updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts              DATETIME NOT NULL,
			kind            TEXT     NOT NULL,
			payload_json    TEXT     NOT NULL,
			correlation_id  TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS dosing_products (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			brand       TEXT     NOT NULL,
			name        TEXT     NOT NULL,
			type        TEXT     NOT NULL CHECK(type IN (
			             'two_part_a','two_part_b','calcium','alkalinity','magnesium',
			             'trace','amino','bacteria','carbon_source','filter_media','other')),
			unit        TEXT     NOT NULL DEFAULT 'mL',
			notes       TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS dosing_schedules (
			id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
			product_id            INTEGER  NOT NULL REFERENCES dosing_products(id) ON DELETE CASCADE,
			amount                REAL     NOT NULL,
			frequency             TEXT     NOT NULL CHECK(frequency IN (
			                       'daily','twice_daily','every_n_days','weekly','as_needed')),
			interval_days         REAL,
			day_of_week           INTEGER,
			enabled               INTEGER  NOT NULL DEFAULT 1,
			last_completed_at     DATETIME,
			next_due_at           DATETIME,
			follow_up_parameter_id INTEGER REFERENCES measurement_parameters(id) ON DELETE SET NULL,
			follow_up_days        INTEGER,
			notes                 TEXT,
			created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS dosing_logs (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			schedule_id  INTEGER  REFERENCES dosing_schedules(id) ON DELETE SET NULL,
			product_id   INTEGER  NOT NULL REFERENCES dosing_products(id),
			amount       REAL     NOT NULL,
			dosed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			notes        TEXT,
			source       TEXT     NOT NULL DEFAULT 'manual' CHECK(source IN ('manual','ai')),
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS maintenance_tasks (
			id                INTEGER  PRIMARY KEY AUTOINCREMENT,
			name              TEXT     NOT NULL,
			description       TEXT,
			frequency         TEXT     NOT NULL CHECK(frequency IN (
			                   'daily','every_n_days','weekly','monthly','as_needed')),
			interval_days     REAL,
			day_of_week       INTEGER,
			enabled           INTEGER  NOT NULL DEFAULT 1,
			last_completed_at DATETIME,
			next_due_at       DATETIME,
			created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS maintenance_logs (
			id           INTEGER  PRIMARY KEY AUTOINCREMENT,
			task_id      INTEGER  NOT NULL REFERENCES maintenance_tasks(id) ON DELETE CASCADE,
			completed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			notes        TEXT,
			source       TEXT     NOT NULL DEFAULT 'manual' CHECK(source IN ('manual','ai')),
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_programs (
			did        TEXT     NOT NULL PRIMARY KEY,
			name       TEXT     NOT NULL,
			type       TEXT     NOT NULL,
			icon       TEXT     NOT NULL DEFAULT '',
			prog       TEXT     NOT NULL,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_program_history (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			did        TEXT     NOT NULL,
			prog       TEXT     NOT NULL,
			changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_alert_events_rule ON alert_events(rule_id, fired_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_dashboard_items_ref ON dashboard_items(item_type, reference_id) WHERE reference_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_dashboard_items_sort ON dashboard_items(sort_order)`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_param ON measurements(parameter_id, measured_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_measurements_date ON measurements(measured_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_livestock_type ON livestock(type)`,
		`CREATE INDEX IF NOT EXISTS idx_livestock_status ON livestock(status)`,
		`CREATE INDEX IF NOT EXISTS idx_livestock_obs ON livestock_observations(livestock_id, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_journal_entries_ts ON journal_entries(ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_journal_entries_category ON journal_entries(category, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_kind_ts ON events(kind, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_dosing_schedules_product ON dosing_schedules(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dosing_schedules_due ON dosing_schedules(next_due_at) WHERE enabled=1`,
		`CREATE INDEX IF NOT EXISTS idx_dosing_logs_schedule ON dosing_logs(schedule_id, dosed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_dosing_logs_product ON dosing_logs(product_id, dosed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_due ON maintenance_tasks(next_due_at) WHERE enabled=1`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_logs_task ON maintenance_logs(task_id, completed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_outlet_program_history_did ON outlet_program_history(did, changed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_device_outlets_device_id ON device_outlets(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_probe_config_device ON probe_config(device_id)`,
	}

	for _, stmt := range tables {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("creating table: %w (%s)", err, firstLine(stmt))
		}
	}
	for _, stmt := range indexes {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("creating index: %w (%s)", err, firstLine(stmt))
		}
	}
	return nil
}

// firstLine returns the first non-empty line of a SQL string, used for error
// messages that point at the offending statement without dumping the whole body.
func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
