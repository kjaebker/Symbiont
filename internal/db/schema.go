package db

import "database/sql"

// CreateSchema creates all DuckDB tables idempotently using CREATE TABLE IF NOT EXISTS.
func CreateSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS probe_readings (
			ts TIMESTAMP NOT NULL,
			probe_did VARCHAR NOT NULL,
			probe_type VARCHAR NOT NULL,
			probe_name VARCHAR NOT NULL,
			value DOUBLE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outlet_states (
			ts TIMESTAMP NOT NULL,
			outlet_did VARCHAR NOT NULL,
			outlet_id INTEGER NOT NULL,
			outlet_name VARCHAR NOT NULL,
			outlet_type VARCHAR NOT NULL,
			state VARCHAR NOT NULL,
			intensity INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS power_events (
			ts TIMESTAMP NOT NULL,
			event_type VARCHAR NOT NULL,
			event_ts BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS controller_meta (
			ts TIMESTAMP NOT NULL,
			serial VARCHAR NOT NULL,
			hostname VARCHAR NOT NULL,
			software VARCHAR NOT NULL,
			hardware VARCHAR NOT NULL,
			controller_type VARCHAR NOT NULL,
			timezone VARCHAR NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// MigrateSchema is a placeholder for future schema migrations.
// It runs after CreateSchema and currently does nothing.
func MigrateSchema(db *sql.DB) error {
	return nil
}
