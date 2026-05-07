package db

import (
	"context"
	"database/sql"
	"fmt"
)

// --- Journal Entries ---

// ListJournalEntries returns journal entries matching the filter, newest first.
func (s *SQLiteDB) ListJournalEntries(ctx context.Context, f JournalFilter) ([]JournalEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, ts, category, sentiment, title, body, source, source_ref, created_at
	          FROM journal_entries WHERE 1=1`
	args := []any{}

	if f.Category != "" {
		query += " AND category = ?"
		args = append(args, f.Category)
	}
	if f.Sentiment != "" {
		query += " AND sentiment = ?"
		args = append(args, f.Sentiment)
	}
	if f.From != nil {
		query += " AND ts >= ?"
		args = append(args, f.From)
	}
	if f.To != nil {
		query += " AND ts <= ?"
		args = append(args, f.To)
	}
	query += " ORDER BY ts DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing journal entries: %w", err)
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(
			&e.ID, &e.TS, &e.Category, &e.Sentiment,
			&e.Title, &e.Body, &e.Source, &e.SourceRef, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning journal entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetJournalEntry returns a single journal entry by ID, or nil if not found.
func (s *SQLiteDB) GetJournalEntry(ctx context.Context, id int64) (*JournalEntry, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, ts, category, sentiment, title, body, source, source_ref, created_at
		 FROM journal_entries WHERE id = ?`, id)
	e := &JournalEntry{}
	err := row.Scan(
		&e.ID, &e.TS, &e.Category, &e.Sentiment,
		&e.Title, &e.Body, &e.Source, &e.SourceRef, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting journal entry %d: %w", id, err)
	}
	return e, nil
}

// InsertJournalEntry inserts a new journal entry and returns its ID.
func (s *SQLiteDB) InsertJournalEntry(ctx context.Context, e JournalEntry) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO journal_entries (category, sentiment, title, body, source, source_ref)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.Category, e.Sentiment, e.Title, e.Body, e.Source, e.SourceRef,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting journal entry: %w", err)
	}
	return res.LastInsertId()
}

// UpdateJournalEntry updates the editable fields of an existing journal entry.
func (s *SQLiteDB) UpdateJournalEntry(ctx context.Context, id int64, e JournalEntry) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE journal_entries SET category = ?, sentiment = ?, title = ?, body = ?
		 WHERE id = ?`,
		e.Category, e.Sentiment, e.Title, e.Body, id,
	)
	if err != nil {
		return fmt.Errorf("updating journal entry %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for journal entry %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("journal entry %d not found", id)
	}
	return nil
}

// DeleteJournalEntry deletes a journal entry by ID.
func (s *SQLiteDB) DeleteJournalEntry(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM journal_entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting journal entry %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for journal entry %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("journal entry %d not found", id)
	}
	return nil
}

// HasJournalEntryBySourceRef reports whether any journal entry exists with the
// given source_ref value. Used to check if the daily prompt has been answered.
func (s *SQLiteDB) HasJournalEntryBySourceRef(ctx context.Context, sourceRef string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM journal_entries WHERE source_ref = ?", sourceRef,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking journal entry by source_ref: %w", err)
	}
	return count > 0, nil
}
