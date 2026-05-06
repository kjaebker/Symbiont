package db

import (
	"context"
	"fmt"
)

// --- Dashboard Items ---

// ListDashboardItems returns all dashboard items ordered by sort_order.
func (s *SQLiteDB) ListDashboardItems(ctx context.Context) ([]DashboardItem, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, item_type, reference_id, label, sort_order, display_mode FROM dashboard_items ORDER BY sort_order",
	)
	if err != nil {
		return nil, fmt.Errorf("listing dashboard items: %w", err)
	}
	defer rows.Close()

	var items []DashboardItem
	for rows.Next() {
		var item DashboardItem
		if err := rows.Scan(&item.ID, &item.ItemType, &item.ReferenceID, &item.Label, &item.SortOrder, &item.DisplayMode); err != nil {
			return nil, fmt.Errorf("scanning dashboard item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ReplaceDashboardLayout replaces the entire dashboard layout in a transaction.
func (s *SQLiteDB) ReplaceDashboardLayout(ctx context.Context, items []DashboardItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM dashboard_items"); err != nil {
		return fmt.Errorf("clearing dashboard items: %w", err)
	}

	for i, item := range items {
		mode := item.DisplayMode
		if mode == "" {
			mode = "normal"
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order, display_mode) VALUES (?, ?, ?, ?, ?)",
			item.ItemType, item.ReferenceID, item.Label, i+1, mode,
		); err != nil {
			return fmt.Errorf("inserting dashboard item %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// AddDashboardItem appends a single item to the dashboard (sort_order = max+1).
func (s *SQLiteDB) AddDashboardItem(ctx context.Context, item DashboardItem) (int64, error) {
	var maxOrder int
	_ = s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order), 0) FROM dashboard_items").Scan(&maxOrder)

	mode := item.DisplayMode
	if mode == "" {
		mode = "normal"
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order, display_mode) VALUES (?, ?, ?, ?, ?)",
		item.ItemType, item.ReferenceID, item.Label, maxOrder+1, mode,
	)
	if err != nil {
		return 0, fmt.Errorf("adding dashboard item: %w", err)
	}
	return res.LastInsertId()
}

// RemoveDashboardItem removes a dashboard item by ID.
func (s *SQLiteDB) RemoveDashboardItem(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM dashboard_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("removing dashboard item %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("dashboard item %d not found", id)
	}
	return nil
}

// RemoveDashboardItemByRef removes a dashboard item by type and reference_id.
func (s *SQLiteDB) RemoveDashboardItemByRef(ctx context.Context, itemType, referenceID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM dashboard_items WHERE item_type = ? AND reference_id = ?",
		itemType, referenceID,
	)
	if err != nil {
		return fmt.Errorf("removing dashboard item by ref %s/%s: %w", itemType, referenceID, err)
	}
	return nil
}
