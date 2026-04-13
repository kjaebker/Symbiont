package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
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
		"SELECT probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning, device_id FROM probe_config WHERE probe_name = ?",
		probeName,
	).Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("getting probe config %s: %w", probeName, err)
	}
	return &c, nil
}

// ListProbeConfigs returns all probe configs ordered by name.
func (s *SQLiteDB) ListProbeConfigs(ctx context.Context) ([]ProbeConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning, device_id FROM probe_config ORDER BY probe_name",
	)
	if err != nil {
		return nil, fmt.Errorf("listing probe configs: %w", err)
	}
	defer rows.Close()

	var configs []ProbeConfig
	for rows.Next() {
		var c ProbeConfig
		if err := rows.Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.DeviceID); err != nil {
			return nil, fmt.Errorf("scanning probe config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertProbeConfig inserts or updates a probe config.
func (s *SQLiteDB) UpsertProbeConfig(ctx context.Context, cfg ProbeConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO probe_config (probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(probe_name) DO UPDATE SET
			display_name = excluded.display_name,
			unit_override = excluded.unit_override,
			min_normal = excluded.min_normal,
			max_normal = excluded.max_normal,
			min_warning = excluded.min_warning,
			max_warning = excluded.max_warning`,
		cfg.ProbeName, cfg.DisplayName, cfg.UnitOverride,
		cfg.MinNormal, cfg.MaxNormal, cfg.MinWarning, cfg.MaxWarning,
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
		"SELECT outlet_id, display_name, icon FROM outlet_config WHERE outlet_id = ?",
		outletID,
	).Scan(&c.OutletID, &c.DisplayName, &c.Icon)
	if err != nil {
		return nil, fmt.Errorf("getting outlet config %s: %w", outletID, err)
	}
	return &c, nil
}

// ListOutletConfigs returns all outlet configs ordered by outlet_id.
func (s *SQLiteDB) ListOutletConfigs(ctx context.Context) ([]OutletConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT outlet_id, display_name, icon FROM outlet_config ORDER BY outlet_id",
	)
	if err != nil {
		return nil, fmt.Errorf("listing outlet configs: %w", err)
	}
	defer rows.Close()

	var configs []OutletConfig
	for rows.Next() {
		var c OutletConfig
		if err := rows.Scan(&c.OutletID, &c.DisplayName, &c.Icon); err != nil {
			return nil, fmt.Errorf("scanning outlet config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertOutletConfig inserts or updates an outlet config.
func (s *SQLiteDB) UpsertOutletConfig(ctx context.Context, cfg OutletConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO outlet_config (outlet_id, display_name, icon)
		VALUES (?, ?, ?)
		ON CONFLICT(outlet_id) DO UPDATE SET
			display_name = excluded.display_name,
			icon = excluded.icon`,
		cfg.OutletID, cfg.DisplayName, cfg.Icon,
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

// --- Devices ---

// ListDevices returns all devices with their linked probe names.
func (s *SQLiteDB) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, device_type, description, brand, model, notes, image_path, outlet_id, created_at, updated_at
		FROM devices ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.DeviceType, &d.Description, &d.Brand, &d.Model, &d.Notes, &d.ImagePath, &d.OutletID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning device: %w", err)
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load probe names for each device.
	for i := range devices {
		probes, err := s.listDeviceProbeNames(ctx, devices[i].ID)
		if err != nil {
			return nil, err
		}
		devices[i].ProbeNames = probes
	}
	return devices, nil
}

// GetDevice returns a single device by ID with its linked probe names.
func (s *SQLiteDB) GetDevice(ctx context.Context, id int64) (*Device, error) {
	var d Device
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, device_type, description, brand, model, notes, image_path, outlet_id, created_at, updated_at
		FROM devices WHERE id = ?`, id,
	).Scan(&d.ID, &d.Name, &d.DeviceType, &d.Description, &d.Brand, &d.Model, &d.Notes, &d.ImagePath, &d.OutletID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting device %d: %w", id, err)
	}
	probes, err := s.listDeviceProbeNames(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.ProbeNames = probes
	return &d, nil
}

// InsertDevice inserts a new device and returns its ID.
func (s *SQLiteDB) InsertDevice(ctx context.Context, d Device) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO devices (name, device_type, description, brand, model, notes, image_path, outlet_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.Name, d.DeviceType, d.Description, d.Brand, d.Model, d.Notes, d.ImagePath, d.OutletID,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting device: %w", err)
	}
	return res.LastInsertId()
}

// UpdateDevice updates an existing device. Returns an error if the device is not found.
func (s *SQLiteDB) UpdateDevice(ctx context.Context, id int64, d Device) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE devices SET name = ?, device_type = ?, description = ?, brand = ?, model = ?, notes = ?, image_path = ?, outlet_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		d.Name, d.DeviceType, d.Description, d.Brand, d.Model, d.Notes, d.ImagePath, d.OutletID, id,
	)
	if err != nil {
		return fmt.Errorf("updating device %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("device %d not found", id)
	}
	return nil
}

// DeleteDevice removes a device by ID. Linked probe_config rows have device_id set to NULL via ON DELETE SET NULL.
func (s *SQLiteDB) DeleteDevice(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting device %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("device %d not found", id)
	}
	return nil
}

// SetProbeDevice sets or clears the device_id on a probe_config row.
// Ensures the probe_config row exists via upsert.
func (s *SQLiteDB) SetProbeDevice(ctx context.Context, probeName string, deviceID *int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO probe_config (probe_name, device_id) VALUES (?, ?)
		ON CONFLICT(probe_name) DO UPDATE SET device_id = excluded.device_id`,
		probeName, deviceID,
	)
	if err != nil {
		return fmt.Errorf("setting probe device for %s: %w", probeName, err)
	}
	return nil
}

// SetDeviceProbes replaces the full probe membership for a device.
// Clears device_id from any probes previously linked, then sets it on the new list.
func (s *SQLiteDB) SetDeviceProbes(ctx context.Context, deviceID int64, probeNames []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing links.
	if _, err := tx.ExecContext(ctx, "UPDATE probe_config SET device_id = NULL WHERE device_id = ?", deviceID); err != nil {
		return fmt.Errorf("clearing probe links for device %d: %w", deviceID, err)
	}

	// Set new links.
	for _, name := range probeNames {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO probe_config (probe_name, device_id) VALUES (?, ?)
			ON CONFLICT(probe_name) DO UPDATE SET device_id = excluded.device_id`,
			name, deviceID,
		); err != nil {
			return fmt.Errorf("linking probe %s to device %d: %w", name, deviceID, err)
		}
	}

	return tx.Commit()
}

// SyncDeviceDisplayNames writes the device name through to linked outlet_config and probe_config rows.
// Both the outlet and all linked probes get the bare device name.
func (s *SQLiteDB) SyncDeviceDisplayNames(ctx context.Context, deviceID int64, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Update outlet display name.
	if _, err := tx.ExecContext(ctx,
		`UPDATE outlet_config SET display_name = ?
		WHERE outlet_id = (SELECT outlet_id FROM devices WHERE id = ? AND outlet_id IS NOT NULL)`,
		name, deviceID,
	); err != nil {
		return fmt.Errorf("syncing outlet display name for device %d: %w", deviceID, err)
	}

	// Update all linked probes with the bare device name.
	if _, err := tx.ExecContext(ctx,
		"UPDATE probe_config SET display_name = ? WHERE device_id = ?",
		name, deviceID,
	); err != nil {
		return fmt.Errorf("syncing probe display names for device %d: %w", deviceID, err)
	}

	return tx.Commit()
}

// GetDeviceByOutletID returns the device linked to an outlet, or nil if none.
func (s *SQLiteDB) GetDeviceByOutletID(ctx context.Context, outletID string) (*Device, error) {
	var d Device
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, device_type, description, brand, model, notes, image_path, outlet_id, created_at, updated_at
		FROM devices WHERE outlet_id = ?`, outletID,
	).Scan(&d.ID, &d.Name, &d.DeviceType, &d.Description, &d.Brand, &d.Model, &d.Notes, &d.ImagePath, &d.OutletID, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting device by outlet %s: %w", outletID, err)
	}
	return &d, nil
}

// GetDeviceByProbeName returns the device linked to a probe, or nil if none.
func (s *SQLiteDB) GetDeviceByProbeName(ctx context.Context, probeName string) (*Device, error) {
	var d Device
	err := s.db.QueryRowContext(ctx,
		`SELECT d.id, d.name, d.device_type, d.description, d.brand, d.model, d.notes, d.image_path, d.outlet_id, d.created_at, d.updated_at
		FROM devices d
		JOIN probe_config pc ON pc.device_id = d.id
		WHERE pc.probe_name = ?`, probeName,
	).Scan(&d.ID, &d.Name, &d.DeviceType, &d.Description, &d.Brand, &d.Model, &d.Notes, &d.ImagePath, &d.OutletID, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting device by probe %s: %w", probeName, err)
	}
	return &d, nil
}

// listDeviceProbeNames returns the probe names linked to a device.
func (s *SQLiteDB) listDeviceProbeNames(ctx context.Context, deviceID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT probe_name FROM probe_config WHERE device_id = ? ORDER BY probe_name", deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing probe names for device %d: %w", deviceID, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scanning probe name: %w", err)
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

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

// --- Measurement Parameters ---

// ListMeasurementParameters returns all parameters ordered by sort_order then name.
func (s *SQLiteDB) ListMeasurementParameters(ctx context.Context) ([]MeasurementParameter, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, canonical_unit, sort_order FROM measurement_parameters ORDER BY sort_order, name",
	)
	if err != nil {
		return nil, fmt.Errorf("listing measurement parameters: %w", err)
	}
	defer rows.Close()

	var params []MeasurementParameter
	for rows.Next() {
		var p MeasurementParameter
		if err := rows.Scan(&p.ID, &p.Name, &p.CanonicalUnit, &p.SortOrder); err != nil {
			return nil, fmt.Errorf("scanning measurement parameter: %w", err)
		}
		params = append(params, p)
	}
	return params, rows.Err()
}

// GetMeasurementParameterByName returns the parameter with the given name, or sql.ErrNoRows.
func (s *SQLiteDB) GetMeasurementParameterByName(ctx context.Context, name string) (*MeasurementParameter, error) {
	var p MeasurementParameter
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, canonical_unit, sort_order FROM measurement_parameters WHERE name = ?",
		name,
	).Scan(&p.ID, &p.Name, &p.CanonicalUnit, &p.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting measurement parameter %q: %w", name, err)
	}
	return &p, nil
}

// InsertMeasurementParameter adds a custom parameter and returns its ID.
func (s *SQLiteDB) InsertMeasurementParameter(ctx context.Context, name, unit string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO measurement_parameters (name, canonical_unit) VALUES (?, ?)",
		name, unit,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting measurement parameter %q: %w", name, err)
	}
	return res.LastInsertId()
}

// --- Measurements ---

// ListMeasurements returns measurements joined with their parameter name and unit.
func (s *SQLiteDB) ListMeasurements(ctx context.Context, f MeasurementFilter) ([]Measurement, error) {
	const base = `
		SELECT m.id, m.measured_at, m.parameter_id, m.value, m.notes, m.source,
		       m.test_kit_ref, m.raw_value, m.created_at, p.name, p.canonical_unit
		FROM measurements m
		JOIN measurement_parameters p ON p.id = m.parameter_id`

	var conditions []string
	var args []any

	if f.ParameterID != nil {
		conditions = append(conditions, "m.parameter_id = ?")
		args = append(args, *f.ParameterID)
	}
	if f.From != nil {
		conditions = append(conditions, "m.measured_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conditions = append(conditions, "m.measured_at <= ?")
		args = append(args, *f.To)
	}

	query := base
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	query += " ORDER BY m.measured_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing measurements: %w", err)
	}
	defer rows.Close()

	var measurements []Measurement
	for rows.Next() {
		var m Measurement
		if err := rows.Scan(
			&m.ID, &m.MeasuredAt, &m.ParameterID, &m.Value, &m.Notes, &m.Source,
			&m.TestKitRef, &m.RawValue, &m.CreatedAt, &m.Parameter, &m.CanonicalUnit,
		); err != nil {
			return nil, fmt.Errorf("scanning measurement: %w", err)
		}
		measurements = append(measurements, m)
	}
	return measurements, rows.Err()
}

// GetMeasurement returns a single measurement by ID, or nil if not found.
func (s *SQLiteDB) GetMeasurement(ctx context.Context, id int64) (*Measurement, error) {
	var m Measurement
	err := s.db.QueryRowContext(ctx, `
		SELECT m.id, m.measured_at, m.parameter_id, m.value, m.notes, m.source,
		       m.test_kit_ref, m.raw_value, m.created_at, p.name, p.canonical_unit
		FROM measurements m
		JOIN measurement_parameters p ON p.id = m.parameter_id
		WHERE m.id = ?`, id,
	).Scan(
		&m.ID, &m.MeasuredAt, &m.ParameterID, &m.Value, &m.Notes, &m.Source,
		&m.TestKitRef, &m.RawValue, &m.CreatedAt, &m.Parameter, &m.CanonicalUnit,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting measurement %d: %w", id, err)
	}
	return &m, nil
}

// InsertMeasurement inserts a new measurement and returns its ID.
func (s *SQLiteDB) InsertMeasurement(ctx context.Context, m Measurement) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO measurements (measured_at, parameter_id, value, notes, source, test_kit_ref, raw_value)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.MeasuredAt, m.ParameterID, m.Value, m.Notes, m.Source, m.TestKitRef, m.RawValue,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting measurement: %w", err)
	}
	return res.LastInsertId()
}

// UpdateMeasurement updates a measurement's mutable fields. Returns an error if not found.
func (s *SQLiteDB) UpdateMeasurement(ctx context.Context, id int64, m Measurement) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE measurements
		SET measured_at = ?, parameter_id = ?, value = ?, notes = ?, test_kit_ref = ?, raw_value = ?
		WHERE id = ?`,
		m.MeasuredAt, m.ParameterID, m.Value, m.Notes, m.TestKitRef, m.RawValue, id,
	)
	if err != nil {
		return fmt.Errorf("updating measurement %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("measurement %d not found", id)
	}
	return nil
}

// DeleteMeasurement removes a measurement by ID. Returns an error if not found.
func (s *SQLiteDB) DeleteMeasurement(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM measurements WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting measurement %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("measurement %d not found", id)
	}
	return nil
}


// MigrateDashboardLayout migrates old hidden/display_order data into dashboard_items.
// It is idempotent: if dashboard_items already has rows, it does nothing.
func (s *SQLiteDB) MigrateDashboardLayout(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dashboard_items").Scan(&count); err != nil {
		return fmt.Errorf("checking dashboard_items count: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning migration transaction: %w", err)
	}
	defer tx.Rollback()

	sortOrder := 1

	// Migrate probes. Fall back to simple query if legacy columns don't exist.
	probeRows, err := tx.QueryContext(ctx,
		"SELECT probe_name FROM probe_config WHERE hidden = 0 ORDER BY display_order, probe_name",
	)
	if err != nil {
		probeRows, err = tx.QueryContext(ctx, "SELECT probe_name FROM probe_config ORDER BY probe_name")
		if err != nil {
			return fmt.Errorf("reading probe configs for migration: %w", err)
		}
	}
	var probeNames []string
	for probeRows.Next() {
		var name string
		if err := probeRows.Scan(&name); err != nil {
			probeRows.Close()
			return fmt.Errorf("scanning probe name: %w", err)
		}
		probeNames = append(probeNames, name)
	}
	probeRows.Close()

	if len(probeNames) > 0 {
		// Add separator for probes.
		label := "Telemetry"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting probe separator: %w", err)
		}
		sortOrder++

		for _, name := range probeNames {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('probe', ?, NULL, ?)",
				name, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting probe dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	// Migrate devices. The hidden/display_order columns may not exist on
	// all production databases, so fall back to a simple query.
	deviceRows, err := tx.QueryContext(ctx,
		"SELECT id FROM devices WHERE hidden = 0 ORDER BY display_order, name",
	)
	if err != nil {
		// Columns don't exist — just select all devices.
		deviceRows, err = tx.QueryContext(ctx, "SELECT id FROM devices ORDER BY name")
		if err != nil {
			return fmt.Errorf("reading devices for migration: %w", err)
		}
	}
	var deviceIDs []string
	for deviceRows.Next() {
		var id int64
		if err := deviceRows.Scan(&id); err != nil {
			deviceRows.Close()
			return fmt.Errorf("scanning device id: %w", err)
		}
		deviceIDs = append(deviceIDs, fmt.Sprintf("%d", id))
	}
	deviceRows.Close()

	if len(deviceIDs) > 0 {
		label := "Equipment"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting device separator: %w", err)
		}
		sortOrder++

		for _, idStr := range deviceIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('device', ?, NULL, ?)",
				idStr, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting device dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	// Migrate outlets. Same fallback as devices.
	outletRows, err := tx.QueryContext(ctx,
		"SELECT outlet_id FROM outlet_config WHERE hidden = 0 ORDER BY display_order, outlet_id",
	)
	if err != nil {
		outletRows, err = tx.QueryContext(ctx, "SELECT outlet_id FROM outlet_config ORDER BY outlet_id")
		if err != nil {
			return fmt.Errorf("reading outlet configs for migration: %w", err)
		}
	}
	var outletIDs []string
	for outletRows.Next() {
		var id string
		if err := outletRows.Scan(&id); err != nil {
			outletRows.Close()
			return fmt.Errorf("scanning outlet id: %w", err)
		}
		outletIDs = append(outletIDs, id)
	}
	outletRows.Close()

	if len(outletIDs) > 0 {
		label := "Controls"
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('separator', NULL, ?, ?)",
			label, sortOrder,
		); err != nil {
			return fmt.Errorf("inserting outlet separator: %w", err)
		}
		sortOrder++

		for _, id := range outletIDs {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO dashboard_items (item_type, reference_id, label, sort_order) VALUES ('outlet', ?, NULL, ?)",
				id, sortOrder,
			); err != nil {
				return fmt.Errorf("inserting outlet dashboard item: %w", err)
			}
			sortOrder++
		}
	}

	return tx.Commit()
}

// --- Livestock ---

// ListLivestock returns livestock items matching the filter, ordered by name.
func (s *SQLiteDB) ListLivestock(ctx context.Context, f LivestockFilter) ([]LivestockItem, error) {
	query := `SELECT id, name, species, type, quantity, status, date_added, notes, image_path, created_at, updated_at
		FROM livestock`

	var conditions []string
	var args []any

	if f.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, f.Type)
	}
	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing livestock: %w", err)
	}
	defer rows.Close()

	var items []LivestockItem
	for rows.Next() {
		var item LivestockItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Species, &item.Type, &item.Quantity, &item.Status, &item.DateAdded, &item.Notes, &item.ImagePath, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning livestock item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []LivestockItem{}
	}
	return items, rows.Err()
}

// ListLivestockSpecies returns distinct non-null species values, ordered alphabetically.
func (s *SQLiteDB) ListLivestockSpecies(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT species FROM livestock WHERE species IS NOT NULL ORDER BY species",
	)
	if err != nil {
		return nil, fmt.Errorf("listing livestock species: %w", err)
	}
	defer rows.Close()

	var species []string
	for rows.Next() {
		var sp string
		if err := rows.Scan(&sp); err != nil {
			return nil, fmt.Errorf("scanning species: %w", err)
		}
		species = append(species, sp)
	}
	if species == nil {
		species = []string{}
	}
	return species, rows.Err()
}

// GetLivestockItem returns a single livestock item by ID.
func (s *SQLiteDB) GetLivestockItem(ctx context.Context, id int64) (*LivestockItem, error) {
	var item LivestockItem
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, species, type, quantity, status, date_added, notes, image_path, created_at, updated_at
		FROM livestock WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.Species, &item.Type, &item.Quantity, &item.Status, &item.DateAdded, &item.Notes, &item.ImagePath, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// InsertLivestockItem inserts a new livestock item and returns its ID.
func (s *SQLiteDB) InsertLivestockItem(ctx context.Context, item LivestockItem) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO livestock (name, species, type, quantity, status, date_added, notes, image_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Name, item.Species, item.Type, item.Quantity, item.Status, item.DateAdded, item.Notes, item.ImagePath,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting livestock item: %w", err)
	}
	return res.LastInsertId()
}

// UpdateLivestockItem updates an existing livestock item.
func (s *SQLiteDB) UpdateLivestockItem(ctx context.Context, id int64, item LivestockItem) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE livestock SET name = ?, species = ?, type = ?, quantity = ?, status = ?, date_added = ?, notes = ?, image_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		item.Name, item.Species, item.Type, item.Quantity, item.Status, item.DateAdded, item.Notes, item.ImagePath, id,
	)
	if err != nil {
		return fmt.Errorf("updating livestock item %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("livestock item %d not found", id)
	}
	return nil
}

// DeleteLivestockItem removes a livestock item by ID.
func (s *SQLiteDB) DeleteLivestockItem(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM livestock WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting livestock item %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("livestock item %d not found", id)
	}
	return nil
}

// ListLivestockObservations returns all observations for a livestock item, newest first.
func (s *SQLiteDB) ListLivestockObservations(ctx context.Context, livestockID int64) ([]LivestockObservation, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, livestock_id, ts, status, note, image_path FROM livestock_observations WHERE livestock_id = ? ORDER BY ts DESC",
		livestockID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing livestock observations: %w", err)
	}
	defer rows.Close()

	var obs []LivestockObservation
	for rows.Next() {
		var o LivestockObservation
		if err := rows.Scan(&o.ID, &o.LivestockID, &o.TS, &o.Status, &o.Note, &o.ImagePath); err != nil {
			return nil, fmt.Errorf("scanning livestock observation: %w", err)
		}
		obs = append(obs, o)
	}
	if obs == nil {
		obs = []LivestockObservation{}
	}
	return obs, rows.Err()
}

// GetLivestockObservation returns a single observation by ID.
func (s *SQLiteDB) GetLivestockObservation(ctx context.Context, id int64) (*LivestockObservation, error) {
	var o LivestockObservation
	err := s.db.QueryRowContext(ctx,
		"SELECT id, livestock_id, ts, status, note, image_path FROM livestock_observations WHERE id = ?",
		id,
	).Scan(&o.ID, &o.LivestockID, &o.TS, &o.Status, &o.Note, &o.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("getting livestock observation: %w", err)
	}
	return &o, nil
}

// UpdateLivestockObservationImagePath sets or clears the image_path for an observation.
func (s *SQLiteDB) UpdateLivestockObservationImagePath(ctx context.Context, id int64, imagePath *string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE livestock_observations SET image_path = ? WHERE id = ?",
		imagePath, id,
	)
	if err != nil {
		return fmt.Errorf("updating livestock observation image: %w", err)
	}
	return nil
}

// InsertLivestockObservation inserts a new observation and returns its ID.
func (s *SQLiteDB) InsertLivestockObservation(ctx context.Context, o LivestockObservation) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO livestock_observations (livestock_id, status, note) VALUES (?, ?, ?)",
		o.LivestockID, o.Status, o.Note,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting livestock observation: %w", err)
	}
	return res.LastInsertId()
}

// --- Tank Profile ---

// GetTankProfile returns the profile for the given section ("display" or "sump").
// Returns nil, nil if no profile has been saved for that section yet.
func (s *SQLiteDB) GetTankProfile(ctx context.Context, section string) (*TankProfile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT section, display_name, volume_gallons, length_in, width_in, height_in,
		        tank_type, manufacturer, model, setup_date, notes, updated_at
		 FROM tank_profile WHERE section = ?`, section)
	p := &TankProfile{}
	err := row.Scan(
		&p.Section, &p.DisplayName, &p.VolumeGallons,
		&p.LengthIn, &p.WidthIn, &p.HeightIn,
		&p.TankType, &p.Manufacturer, &p.Model,
		&p.SetupDate, &p.Notes, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting tank profile %q: %w", section, err)
	}
	return p, nil
}

// UpsertTankProfile inserts or replaces the tank profile for the given section.
func (s *SQLiteDB) UpsertTankProfile(ctx context.Context, p TankProfile) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tank_profile
		    (section, display_name, volume_gallons, length_in, width_in, height_in,
		     tank_type, manufacturer, model, setup_date, notes, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(section) DO UPDATE SET
		    display_name   = excluded.display_name,
		    volume_gallons = excluded.volume_gallons,
		    length_in      = excluded.length_in,
		    width_in       = excluded.width_in,
		    height_in      = excluded.height_in,
		    tank_type      = excluded.tank_type,
		    manufacturer   = excluded.manufacturer,
		    model          = excluded.model,
		    setup_date     = excluded.setup_date,
		    notes          = excluded.notes,
		    updated_at     = CURRENT_TIMESTAMP`,
		p.Section, p.DisplayName, p.VolumeGallons,
		p.LengthIn, p.WidthIn, p.HeightIn,
		p.TankType, p.Manufacturer, p.Model,
		p.SetupDate, p.Notes,
	)
	if err != nil {
		return fmt.Errorf("upserting tank profile %q: %w", p.Section, err)
	}
	return nil
}

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

// --- Audit Events ---

// InsertAuditEvent inserts a row into the events table.
func (s *SQLiteDB) InsertAuditEvent(ctx context.Context, e AuditEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (ts, kind, payload_json, correlation_id) VALUES (?, ?, ?, ?)`,
		e.TS, e.Kind, e.PayloadJSON, e.CorrelationID,
	)
	if err != nil {
		return fmt.Errorf("inserting audit event kind=%s: %w", e.Kind, err)
	}
	return nil
}

// ListAuditEvents returns rows from the events table filtered by AuditFilter,
// ordered by ts DESC. Limit defaults to 100 and is capped at 500.
func (s *SQLiteDB) ListAuditEvents(ctx context.Context, f AuditFilter) ([]AuditEvent, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	query := `SELECT id, ts, kind, payload_json, correlation_id FROM events WHERE 1=1`
	var args []any

	if f.Kind != "" {
		query += ` AND kind = ?`
		args = append(args, f.Kind)
	}
	if f.Since != nil {
		query += ` AND ts >= ?`
		args = append(args, f.Since)
	}
	if f.CorrelationID != "" {
		query += ` AND correlation_id = ?`
		args = append(args, f.CorrelationID)
	}
	if f.InitiatedBy != "" {
		query += ` AND json_extract(payload_json, '$.data.initiated_by') = ?`
		args = append(args, f.InitiatedBy)
	}
	query += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}
	defer rows.Close()

	var out []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.TS, &e.Kind, &e.PayloadJSON, &e.CorrelationID); err != nil {
			return nil, fmt.Errorf("scanning audit event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
