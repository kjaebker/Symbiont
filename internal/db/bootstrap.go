package db

import (
	"context"
	"fmt"
)

// EnsureDefaultToken checks if any auth tokens exist. If none, it generates
// a default token and returns it. The bool indicates whether a new token was created.
func (s *SQLiteDB) EnsureDefaultToken(ctx context.Context) (string, bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth_tokens").Scan(&count); err != nil {
		return "", false, fmt.Errorf("checking token count: %w", err)
	}

	if count > 0 {
		return "", false, nil
	}

	token, err := s.InsertToken(ctx, "default")
	if err != nil {
		return "", false, fmt.Errorf("creating default token: %w", err)
	}

	return token, true, nil
}
