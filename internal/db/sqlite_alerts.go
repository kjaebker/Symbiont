package db

import (
	"context"
	"fmt"
)

// --- Alert Rules ---

// ListEnabledAlertRules returns all enabled alert rules.
func (s *SQLiteDB) ListEnabledAlertRules(ctx context.Context) ([]AlertRule, error) {
	return s.listAlertRules(ctx, "SELECT id, probe_name, condition, threshold_low, threshold_high, severity, cooldown_minutes, enabled, created_at FROM alert_rules WHERE enabled = 1 ORDER BY id")
}

// ListAlertRules returns all alert rules.
func (s *SQLiteDB) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	return s.listAlertRules(ctx, "SELECT id, probe_name, condition, threshold_low, threshold_high, severity, cooldown_minutes, enabled, created_at FROM alert_rules ORDER BY id")
}

func (s *SQLiteDB) listAlertRules(ctx context.Context, query string) ([]AlertRule, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing alert rules: %w", err)
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.ProbeName, &r.Condition, &r.ThresholdLow, &r.ThresholdHigh, &r.Severity, &r.CooldownMinutes, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning alert rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// InsertAlertRule inserts a new alert rule and returns its ID.
func (s *SQLiteDB) InsertAlertRule(ctx context.Context, rule AlertRule) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_rules (probe_name, condition, threshold_low, threshold_high, severity, cooldown_minutes, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rule.ProbeName, rule.Condition, rule.ThresholdLow, rule.ThresholdHigh, rule.Severity, rule.CooldownMinutes, rule.Enabled,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting alert rule: %w", err)
	}
	return res.LastInsertId()
}

// UpdateAlertRule updates an existing alert rule.
func (s *SQLiteDB) UpdateAlertRule(ctx context.Context, id int64, rule AlertRule) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE alert_rules SET probe_name = ?, condition = ?, threshold_low = ?, threshold_high = ?, severity = ?, cooldown_minutes = ?, enabled = ? WHERE id = ?`,
		rule.ProbeName, rule.Condition, rule.ThresholdLow, rule.ThresholdHigh, rule.Severity, rule.CooldownMinutes, rule.Enabled, id,
	)
	if err != nil {
		return fmt.Errorf("updating alert rule %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("alert rule %d not found", id)
	}
	return nil
}

// DeleteAlertRule removes an alert rule by ID.
func (s *SQLiteDB) DeleteAlertRule(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM alert_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting alert rule %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("alert rule %d not found", id)
	}
	return nil
}

// --- Alert Events ---

// InsertAlertEvent inserts a new alert event and returns its ID.
func (s *SQLiteDB) InsertAlertEvent(ctx context.Context, ruleID int64, peakValue float64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_events (rule_id, peak_value, notified) VALUES (?, ?, 0)`,
		ruleID, peakValue,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting alert event: %w", err)
	}
	return res.LastInsertId()
}

// ClearAlertEvent sets the cleared_at timestamp on an alert event.
func (s *SQLiteDB) ClearAlertEvent(ctx context.Context, eventID int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE alert_events SET cleared_at = CURRENT_TIMESTAMP WHERE id = ?",
		eventID,
	)
	if err != nil {
		return fmt.Errorf("clearing alert event %d: %w", eventID, err)
	}
	return nil
}

// UpdateAlertEventPeak updates the peak value and marks as notified.
func (s *SQLiteDB) UpdateAlertEventPeak(ctx context.Context, eventID int64, peakValue float64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE alert_events SET peak_value = ?, notified = 1 WHERE id = ?",
		peakValue, eventID,
	)
	if err != nil {
		return fmt.Errorf("updating alert event peak %d: %w", eventID, err)
	}
	return nil
}

// MarkAlertEventNotified marks an alert event as having sent a notification.
func (s *SQLiteDB) MarkAlertEventNotified(ctx context.Context, eventID int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE alert_events SET notified = 1 WHERE id = ?",
		eventID,
	)
	if err != nil {
		return fmt.Errorf("marking alert event notified %d: %w", eventID, err)
	}
	return nil
}

// ListAlertEvents returns recent alert events, optionally filtered.
func (s *SQLiteDB) ListAlertEvents(ctx context.Context, ruleID *int64, activeOnly bool, limit int) ([]AlertEvent, error) {
	query := `SELECT e.id, e.rule_id, e.fired_at, e.cleared_at, e.peak_value, e.notified, r.probe_name, r.severity
		FROM alert_events e
		LEFT JOIN alert_rules r ON e.rule_id = r.id`

	var conditions []string
	var args []any

	if ruleID != nil {
		conditions = append(conditions, "e.rule_id = ?")
		args = append(args, *ruleID)
	}
	if activeOnly {
		conditions = append(conditions, "e.cleared_at IS NULL")
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}

	query += " ORDER BY e.fired_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing alert events: %w", err)
	}
	defer rows.Close()

	var events []AlertEvent
	for rows.Next() {
		var e AlertEvent
		if err := rows.Scan(&e.ID, &e.RuleID, &e.FiredAt, &e.ClearedAt, &e.PeakValue, &e.Notified, &e.ProbeName, &e.Severity); err != nil {
			return nil, fmt.Errorf("scanning alert event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
