package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// --- Auth Tokens ---

// ValidateToken checks if a token exists and returns its ID.
func (s *SQLiteDB) ValidateToken(ctx context.Context, token string) (bool, int64) {
	var id int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM auth_tokens WHERE token = ?", token).Scan(&id)
	if err != nil {
		return false, 0
	}
	return true, id
}

// TouchToken updates the last_used timestamp for a token.
func (s *SQLiteDB) TouchToken(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE auth_tokens SET last_used = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("touching token %d: %w", id, err)
	}
	return nil
}

// InsertToken generates a random 32-byte token, inserts it, and returns the hex-encoded token string.
func (s *SQLiteDB) InsertToken(ctx context.Context, label string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	token := hex.EncodeToString(b)

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO auth_tokens (token, label) VALUES (?, ?)",
		token, label,
	)
	if err != nil {
		return "", fmt.Errorf("inserting token: %w", err)
	}
	return token, nil
}

// ListTokens returns all tokens (without the token value itself).
func (s *SQLiteDB) ListTokens(ctx context.Context) ([]AuthToken, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, label, created_at, last_used FROM auth_tokens ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing tokens: %w", err)
	}
	defer rows.Close()

	var tokens []AuthToken
	for rows.Next() {
		var t AuthToken
		if err := rows.Scan(&t.ID, &t.Label, &t.CreatedAt, &t.LastUsed); err != nil {
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

// --- Probe Config ---

// GetProbeConfig returns the config for a single probe.
func (s *SQLiteDB) GetProbeConfig(ctx context.Context, probeName string) (*ProbeConfig, error) {
	var c ProbeConfig
	err := s.db.QueryRowContext(ctx,
		"SELECT probe_name, display_name, unit_override, display_order, min_normal, max_normal, min_warning, max_warning, hidden FROM probe_config WHERE probe_name = ?",
		probeName,
	).Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.DisplayOrder, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.Hidden)
	if err != nil {
		return nil, fmt.Errorf("getting probe config %s: %w", probeName, err)
	}
	return &c, nil
}

// ListProbeConfigs returns all probe configs ordered by display_order.
func (s *SQLiteDB) ListProbeConfigs(ctx context.Context) ([]ProbeConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT probe_name, display_name, unit_override, display_order, min_normal, max_normal, min_warning, max_warning, hidden FROM probe_config ORDER BY display_order, probe_name",
	)
	if err != nil {
		return nil, fmt.Errorf("listing probe configs: %w", err)
	}
	defer rows.Close()

	var configs []ProbeConfig
	for rows.Next() {
		var c ProbeConfig
		if err := rows.Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.DisplayOrder, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.Hidden); err != nil {
			return nil, fmt.Errorf("scanning probe config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertProbeConfig inserts or updates a probe config.
func (s *SQLiteDB) UpsertProbeConfig(ctx context.Context, cfg ProbeConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO probe_config (probe_name, display_name, unit_override, display_order, min_normal, max_normal, min_warning, max_warning, hidden)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(probe_name) DO UPDATE SET
			display_name = excluded.display_name,
			unit_override = excluded.unit_override,
			display_order = excluded.display_order,
			min_normal = excluded.min_normal,
			max_normal = excluded.max_normal,
			min_warning = excluded.min_warning,
			max_warning = excluded.max_warning,
			hidden = excluded.hidden`,
		cfg.ProbeName, cfg.DisplayName, cfg.UnitOverride, cfg.DisplayOrder,
		cfg.MinNormal, cfg.MaxNormal, cfg.MinWarning, cfg.MaxWarning, cfg.Hidden,
	)
	if err != nil {
		return fmt.Errorf("upserting probe config %s: %w", cfg.ProbeName, err)
	}
	return nil
}

// --- Outlet Config ---

// GetOutletConfig returns the config for a single outlet.
func (s *SQLiteDB) GetOutletConfig(ctx context.Context, outletID string) (*OutletConfig, error) {
	var c OutletConfig
	err := s.db.QueryRowContext(ctx,
		"SELECT outlet_id, display_name, display_order, icon, hidden FROM outlet_config WHERE outlet_id = ?",
		outletID,
	).Scan(&c.OutletID, &c.DisplayName, &c.DisplayOrder, &c.Icon, &c.Hidden)
	if err != nil {
		return nil, fmt.Errorf("getting outlet config %s: %w", outletID, err)
	}
	return &c, nil
}

// ListOutletConfigs returns all outlet configs ordered by display_order.
func (s *SQLiteDB) ListOutletConfigs(ctx context.Context) ([]OutletConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT outlet_id, display_name, display_order, icon, hidden FROM outlet_config ORDER BY display_order, outlet_id",
	)
	if err != nil {
		return nil, fmt.Errorf("listing outlet configs: %w", err)
	}
	defer rows.Close()

	var configs []OutletConfig
	for rows.Next() {
		var c OutletConfig
		if err := rows.Scan(&c.OutletID, &c.DisplayName, &c.DisplayOrder, &c.Icon, &c.Hidden); err != nil {
			return nil, fmt.Errorf("scanning outlet config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertOutletConfig inserts or updates an outlet config.
func (s *SQLiteDB) UpsertOutletConfig(ctx context.Context, cfg OutletConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO outlet_config (outlet_id, display_name, display_order, icon, hidden)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(outlet_id) DO UPDATE SET
			display_name = excluded.display_name,
			display_order = excluded.display_order,
			icon = excluded.icon,
			hidden = excluded.hidden`,
		cfg.OutletID, cfg.DisplayName, cfg.DisplayOrder, cfg.Icon, cfg.Hidden,
	)
	if err != nil {
		return fmt.Errorf("upserting outlet config %s: %w", cfg.OutletID, err)
	}
	return nil
}

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

// --- Outlet Event Log ---

// InsertOutletEvent inserts an outlet state change event.
func (s *SQLiteDB) InsertOutletEvent(ctx context.Context, e OutletEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO outlet_event_log (outlet_id, outlet_name, from_state, to_state, initiated_by)
		VALUES (?, ?, ?, ?, ?)`,
		e.OutletID, e.OutletName, e.FromState, e.ToState, e.InitiatedBy,
	)
	if err != nil {
		return fmt.Errorf("inserting outlet event: %w", err)
	}
	return nil
}

// ListOutletEvents returns recent outlet events, optionally filtered by outlet ID.
func (s *SQLiteDB) ListOutletEvents(ctx context.Context, outletID string, limit int) ([]OutletEvent, error) {
	var query string
	var args []any

	if outletID != "" {
		query = "SELECT id, ts, outlet_id, outlet_name, from_state, to_state, initiated_by FROM outlet_event_log WHERE outlet_id = ? ORDER BY ts DESC LIMIT ?"
		args = []any{outletID, limit}
	} else {
		query = "SELECT id, ts, outlet_id, outlet_name, from_state, to_state, initiated_by FROM outlet_event_log ORDER BY ts DESC LIMIT ?"
		args = []any{limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing outlet events: %w", err)
	}
	defer rows.Close()

	var events []OutletEvent
	for rows.Next() {
		var e OutletEvent
		if err := rows.Scan(&e.ID, &e.TS, &e.OutletID, &e.OutletName, &e.FromState, &e.ToState, &e.InitiatedBy); err != nil {
			return nil, fmt.Errorf("scanning outlet event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// --- Backup Jobs ---

// InsertBackupJob inserts a new backup job record and returns its ID.
func (s *SQLiteDB) InsertBackupJob(ctx context.Context, job BackupJob) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO backup_jobs (status, path, size_bytes, error) VALUES (?, ?, ?, ?)`,
		job.Status, job.Path, job.SizeBytes, job.Error,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting backup job: %w", err)
	}
	return res.LastInsertId()
}

// UpdateBackupJob updates an existing backup job's status and error.
func (s *SQLiteDB) UpdateBackupJob(ctx context.Context, id int64, status string, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE backup_jobs SET status = ?, error = ? WHERE id = ?",
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("updating backup job %d: %w", id, err)
	}
	return nil
}

// ListBackupJobs returns recent backup jobs.
func (s *SQLiteDB) ListBackupJobs(ctx context.Context, limit int) ([]BackupJob, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, ts, status, path, size_bytes, error FROM backup_jobs ORDER BY ts DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing backup jobs: %w", err)
	}
	defer rows.Close()

	var jobs []BackupJob
	for rows.Next() {
		var j BackupJob
		if err := rows.Scan(&j.ID, &j.TS, &j.Status, &j.Path, &j.SizeBytes, &j.Error); err != nil {
			return nil, fmt.Errorf("scanning backup job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
