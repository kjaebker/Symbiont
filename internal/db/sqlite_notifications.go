package db

import (
	"context"
	"fmt"
)

// --- Notification Targets ---

// ListNotificationTargets returns all notification targets.
func (s *SQLiteDB) ListNotificationTargets(ctx context.Context) ([]NotificationTarget, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, type, config, label, enabled FROM notification_targets ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("listing notification targets: %w", err)
	}
	defer rows.Close()

	var targets []NotificationTarget
	for rows.Next() {
		var t NotificationTarget
		if err := rows.Scan(&t.ID, &t.Type, &t.Config, &t.Label, &t.Enabled); err != nil {
			return nil, fmt.Errorf("scanning notification target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// ListEnabledNotificationTargets returns enabled notification targets of a given type.
func (s *SQLiteDB) ListEnabledNotificationTargets(ctx context.Context, targetType string) ([]NotificationTarget, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, type, config, label, enabled FROM notification_targets WHERE enabled = 1 AND type = ? ORDER BY id",
		targetType,
	)
	if err != nil {
		return nil, fmt.Errorf("listing enabled notification targets: %w", err)
	}
	defer rows.Close()

	var targets []NotificationTarget
	for rows.Next() {
		var t NotificationTarget
		if err := rows.Scan(&t.ID, &t.Type, &t.Config, &t.Label, &t.Enabled); err != nil {
			return nil, fmt.Errorf("scanning notification target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// UpsertNotificationTarget inserts or updates a notification target.
func (s *SQLiteDB) UpsertNotificationTarget(ctx context.Context, t NotificationTarget) (int64, error) {
	if t.ID > 0 {
		_, err := s.db.ExecContext(ctx,
			"UPDATE notification_targets SET type = ?, config = ?, label = ?, enabled = ? WHERE id = ?",
			t.Type, t.Config, t.Label, t.Enabled, t.ID,
		)
		if err != nil {
			return 0, fmt.Errorf("updating notification target %d: %w", t.ID, err)
		}
		return t.ID, nil
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO notification_targets (type, config, label, enabled) VALUES (?, ?, ?, ?)",
		t.Type, t.Config, t.Label, t.Enabled,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting notification target: %w", err)
	}
	return res.LastInsertId()
}

// DeleteNotificationTarget removes a notification target by ID.
func (s *SQLiteDB) DeleteNotificationTarget(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM notification_targets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting notification target %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("notification target %d not found", id)
	}
	return nil
}
