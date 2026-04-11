package db

import (
	"context"
	"fmt"
)

// SeedMeasurementParameters inserts the default water chemistry parameter definitions
// using INSERT OR IGNORE, so it is safe to call on every startup.
func (s *SQLiteDB) SeedMeasurementParameters(ctx context.Context) error {
	params := []struct {
		name  string
		unit  string
		order int
	}{
		{"Alkalinity", "dKH", 1},
		{"Calcium", "ppm", 2},
		{"Magnesium", "ppm", 3},
		{"Nitrate", "ppm", 4},
		{"Nitrite", "ppm", 5},
		{"Phosphate", "ppb", 6},
		{"Salinity", "ppt", 7},
		{"pH", "", 8},
		{"Ammonia", "ppm", 9},
		{"Silicate", "ppm", 10},
	}

	for _, p := range params {
		if _, err := s.db.ExecContext(ctx,
			"INSERT OR IGNORE INTO measurement_parameters (name, canonical_unit, sort_order) VALUES (?, ?, ?)",
			p.name, p.unit, p.order,
		); err != nil {
			return fmt.Errorf("seeding measurement parameter %q: %w", p.name, err)
		}
	}
	return nil
}

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
