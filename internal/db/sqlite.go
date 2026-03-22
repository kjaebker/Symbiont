package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteDB wraps a *sql.DB connection to a SQLite database.
type SQLiteDB struct {
	db *sql.DB
}

// DB returns the underlying *sql.DB.
func (s *SQLiteDB) DB() *sql.DB {
	return s.db
}

// OpenSQLite opens a SQLite database at the given path, runs PRAGMAs, and creates
// the schema if it does not already exist.
func OpenSQLite(path string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %s: %w", path, err)
	}

	// SQLite only supports one writer at a time. Limiting to one connection
	// also ensures :memory: databases share state across goroutines.
	db.SetMaxOpenConns(1)

	if err := CreateSQLiteSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating sqlite schema: %w", err)
	}

	return &SQLiteDB{db: db}, nil
}

// Close closes the underlying database connection.
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
