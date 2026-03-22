package db

import "database/sql"

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
			display_order   INTEGER NOT NULL DEFAULT 999,
			min_normal      REAL,
			max_normal      REAL,
			min_warning     REAL,
			max_warning     REAL,
			hidden          INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_config (
			outlet_id       TEXT PRIMARY KEY,
			display_name    TEXT,
			display_order   INTEGER NOT NULL DEFAULT 999,
			icon            TEXT,
			hidden          INTEGER NOT NULL DEFAULT 0
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
		`CREATE TABLE IF NOT EXISTS outlet_event_log (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			outlet_id       TEXT     NOT NULL,
			outlet_name     TEXT,
			from_state      TEXT,
			to_state        TEXT     NOT NULL,
			initiated_by    TEXT     NOT NULL CHECK(initiated_by IN ('ui','cli','mcp','api','apex'))
		)`,
		`CREATE TABLE IF NOT EXISTS backup_jobs (
			id          INTEGER  PRIMARY KEY AUTOINCREMENT,
			ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status      TEXT     NOT NULL CHECK(status IN ('success','failed')),
			path        TEXT,
			size_bytes  INTEGER,
			error       TEXT
		)`,
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_alert_events_rule ON alert_events(rule_id, fired_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_outlet_event_log_ts ON outlet_event_log(ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_outlet_event_log_outlet ON outlet_event_log(outlet_id, ts DESC)`,
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
	return nil
}
