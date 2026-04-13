package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateSQLiteSchema creates all SQLite tables and indexes idempotently.
func CreateSQLiteSchema(db *sql.DB) error {
	// Enable WAL mode and foreign keys.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}

	tables := []string{
		`CREATE TABLE IF NOT EXISTS auth_tokens (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			token       TEXT     NOT NULL UNIQUE,
			label       TEXT,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used   DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS probe_config (
			probe_name      TEXT PRIMARY KEY,
			display_name    TEXT,
			unit_override   TEXT,
			min_normal      REAL,
			max_normal      REAL,
			min_warning     REAL,
			max_warning     REAL,
			device_id       INTEGER REFERENCES devices(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_config (
			outlet_id       TEXT PRIMARY KEY,
			display_name    TEXT,
			icon            TEXT
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
		`CREATE TABLE IF NOT EXISTS notification_targets (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			type        TEXT     NOT NULL,
			config      TEXT     NOT NULL,
			label       TEXT,
			enabled     INTEGER  NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS alert_events (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			rule_id         INTEGER  NOT NULL REFERENCES alert_rules(id),
			fired_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cleared_at      DATETIME,
			peak_value      REAL,
			notified        INTEGER  NOT NULL DEFAULT 0
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
			item_type    TEXT    NOT NULL CHECK(item_type IN ('probe','outlet','device','separator')),
			reference_id TEXT,
			label        TEXT,
			sort_order   INTEGER NOT NULL
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
			note         TEXT
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
		`CREATE TABLE IF NOT EXISTS events (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts              DATETIME NOT NULL,
			kind            TEXT     NOT NULL,
			payload_json    TEXT     NOT NULL,
			correlation_id  TEXT
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
	}

	for _, stmt := range tables {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Migrations: add columns to existing tables. Each migration catches
	// "duplicate column" to be idempotent.
	migrations := []string{
		`ALTER TABLE probe_config ADD COLUMN device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL`,
		`ALTER TABLE livestock_observations ADD COLUMN image_path TEXT`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			if !isDuplicateColumn(err) {
				return err
			}
		}
	}

	// Structural migrations that require table recreation.
	if err := migrateDashboardItemTypes(db); err != nil {
		return err
	}
	if err := migrateDashboardItemTypesV2(db); err != nil {
		return err
	}

	// Add display_mode after structural migrations so the table-recreation
	// migrations don't have to account for this column.
	if _, err := db.Exec(`ALTER TABLE dashboard_items ADD COLUMN display_mode TEXT NOT NULL DEFAULT 'normal'`); err != nil {
		if !isDuplicateColumn(err) {
			return fmt.Errorf("adding display_mode column: %w", err)
		}
	}

	// Post-migration indexes (depend on columns added above).
	postIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_probe_config_device ON probe_config(device_id)`,
	}
	for _, stmt := range postIndexes {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

// migrateDashboardItemTypesV2 expands the dashboard_items CHECK constraint to
// include 'measurement'.
func migrateDashboardItemTypesV2(db *sql.DB) error {
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='dashboard_items'`).Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("querying dashboard_items schema: %w", err)
	}
	if strings.Contains(createSQL, "'measurement'") {
		return nil // already migrated
	}

	stmts := []string{
		`CREATE TABLE dashboard_items_new (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			item_type    TEXT    NOT NULL CHECK(item_type IN ('probe','outlet','device','separator','feed_mode','measurement')),
			reference_id TEXT,
			label        TEXT,
			sort_order   INTEGER NOT NULL
		)`,
		`INSERT INTO dashboard_items_new SELECT * FROM dashboard_items`,
		`DROP TABLE dashboard_items`,
		`ALTER TABLE dashboard_items_new RENAME TO dashboard_items`,
		`CREATE UNIQUE INDEX idx_dashboard_items_ref ON dashboard_items(item_type, reference_id) WHERE reference_id IS NOT NULL`,
		`CREATE INDEX idx_dashboard_items_sort ON dashboard_items(sort_order)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrating dashboard_items item_type constraint v2: %w", err)
		}
	}
	return nil
}

// isDuplicateColumn checks if a SQLite error is a "duplicate column name" error.
func isDuplicateColumn(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name")
}

// migrateDashboardItemTypes expands the dashboard_items CHECK constraint to
// include 'feed_mode'. SQLite does not support ALTER TABLE to change constraints,
// so this recreates the table when the migration has not yet been applied.
func migrateDashboardItemTypes(db *sql.DB) error {
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='dashboard_items'`).Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("querying dashboard_items schema: %w", err)
	}
	if strings.Contains(createSQL, "'feed_mode'") {
		return nil // already migrated
	}

	stmts := []string{
		`CREATE TABLE dashboard_items_new (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			item_type    TEXT    NOT NULL CHECK(item_type IN ('probe','outlet','device','separator','feed_mode')),
			reference_id TEXT,
			label        TEXT,
			sort_order   INTEGER NOT NULL
		)`,
		`INSERT INTO dashboard_items_new SELECT * FROM dashboard_items`,
		`DROP TABLE dashboard_items`,
		`ALTER TABLE dashboard_items_new RENAME TO dashboard_items`,
		`CREATE UNIQUE INDEX idx_dashboard_items_ref ON dashboard_items(item_type, reference_id) WHERE reference_id IS NOT NULL`,
		`CREATE INDEX idx_dashboard_items_sort ON dashboard_items(sort_order)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrating dashboard_items item_type constraint: %w", err)
		}
	}
	return nil
}

