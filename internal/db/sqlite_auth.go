package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// --- Auth Tokens ---

// ValidateToken checks if a token exists and returns its ID.
func (s *SQLiteDB) ValidateToken(ctx context.Context, token string) (bool, *TokenAuth) {
	var t TokenAuth
	err := s.db.QueryRowContext(ctx,
		"SELECT id, COALESCE(label,''), COALESCE(scope,'admin') FROM auth_tokens WHERE token = ?", token,
	).Scan(&t.ID, &t.Label, &t.Scope)
	if err != nil {
		return false, nil
	}
	return true, &t
}

// TouchToken updates the last_used timestamp for a token.
func (s *SQLiteDB) TouchToken(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE auth_tokens SET last_used = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("touching token %d: %w", id, err)
	}
	return nil
}

// UpdateTokenScope changes the scope of an existing token.
// Only the scope column is updated — the token value is immutable.
func (s *SQLiteDB) UpdateTokenScope(ctx context.Context, id int64, scope string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE auth_tokens SET scope = ? WHERE id = ?", scope, id)
	if err != nil {
		return fmt.Errorf("updating token %d scope: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token %d not found", id)
	}
	return nil
}

// InsertToken generates a random 32-byte token with admin scope and returns it.
// Existing callers (bootstrap, tests) continue to work unchanged.
func (s *SQLiteDB) InsertToken(ctx context.Context, label string) (string, error) {
	return s.InsertTokenWithScope(ctx, label, "admin")
}

// InsertTokenWithScope generates a random 32-byte token with a specific scope.
// scope must be one of: read, control, admin.
func (s *SQLiteDB) InsertTokenWithScope(ctx context.Context, label, scope string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	token := hex.EncodeToString(b)

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO auth_tokens (token, label, scope) VALUES (?, ?, ?)",
		token, label, scope,
	)
	if err != nil {
		return "", fmt.Errorf("inserting token: %w", err)
	}
	return token, nil
}

// ListTokens returns all tokens (without the token value itself).
func (s *SQLiteDB) ListTokens(ctx context.Context) ([]AuthToken, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, COALESCE(label,''), COALESCE(scope,'admin'), created_at, last_used FROM auth_tokens ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing tokens: %w", err)
	}
	defer rows.Close()

	var tokens []AuthToken
	for rows.Next() {
		var t AuthToken
		if err := rows.Scan(&t.ID, &t.Label, &t.Scope, &t.CreatedAt, &t.LastUsed); err != nil {
			return nil, fmt.Errorf("scanning token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// DeleteToken removes a token by ID.
func (s *SQLiteDB) DeleteToken(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting token %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token %d not found", id)
	}
	return nil
}
