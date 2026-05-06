package db

import (
	"context"
	"fmt"
)

// --- Probe Config ---

// GetProbeConfig returns the config for a single probe.
func (s *SQLiteDB) GetProbeConfig(ctx context.Context, probeName string) (*ProbeConfig, error) {
	var c ProbeConfig
	err := s.db.QueryRowContext(ctx,
		"SELECT probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning, device_id, input_category, on_label, off_label, ok_value, is_binary, hidden FROM probe_config WHERE probe_name = ?",
		probeName,
	).Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.DeviceID, &c.InputCategory, &c.OnLabel, &c.OffLabel, &c.OkValue, &c.IsBinary, &c.Hidden)
	if err != nil {
		return nil, fmt.Errorf("getting probe config %s: %w", probeName, err)
	}
	return &c, nil
}

// ListProbeConfigs returns all probe configs ordered by name.
func (s *SQLiteDB) ListProbeConfigs(ctx context.Context) ([]ProbeConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning, device_id, input_category, on_label, off_label, ok_value, is_binary, hidden FROM probe_config ORDER BY probe_name",
	)
	if err != nil {
		return nil, fmt.Errorf("listing probe configs: %w", err)
	}
	defer rows.Close()

	var configs []ProbeConfig
	for rows.Next() {
		var c ProbeConfig
		if err := rows.Scan(&c.ProbeName, &c.DisplayName, &c.UnitOverride, &c.MinNormal, &c.MaxNormal, &c.MinWarning, &c.MaxWarning, &c.DeviceID, &c.InputCategory, &c.OnLabel, &c.OffLabel, &c.OkValue, &c.IsBinary, &c.Hidden); err != nil {
			return nil, fmt.Errorf("scanning probe config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertProbeConfig inserts or updates a probe config.
func (s *SQLiteDB) UpsertProbeConfig(ctx context.Context, cfg ProbeConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO probe_config (probe_name, display_name, unit_override, min_normal, max_normal, min_warning, max_warning, input_category, on_label, off_label, ok_value, is_binary, hidden)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(probe_name) DO UPDATE SET
			display_name = excluded.display_name,
			unit_override = excluded.unit_override,
			min_normal = excluded.min_normal,
			max_normal = excluded.max_normal,
			min_warning = excluded.min_warning,
			max_warning = excluded.max_warning,
			input_category = excluded.input_category,
			on_label = excluded.on_label,
			off_label = excluded.off_label,
			ok_value = excluded.ok_value,
			is_binary = excluded.is_binary,
			hidden = excluded.hidden`,
		cfg.ProbeName, cfg.DisplayName, cfg.UnitOverride,
		cfg.MinNormal, cfg.MaxNormal, cfg.MinWarning, cfg.MaxWarning,
		cfg.InputCategory, cfg.OnLabel, cfg.OffLabel, cfg.OkValue, cfg.IsBinary, cfg.Hidden,
	)
	if err != nil {
		return fmt.Errorf("upserting probe config %s: %w", cfg.ProbeName, err)
	}
	return nil
}

// InitProbeConfig inserts classification defaults for a probe only if no row
// exists yet. Existing user config is never overwritten.
func (s *SQLiteDB) InitProbeConfig(ctx context.Context, cfg ProbeConfig) error {
	if cfg.InputCategory == "" {
		cfg.InputCategory = "probe"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO probe_config (probe_name, input_category, on_label, off_label, ok_value, is_binary, hidden)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cfg.ProbeName, cfg.InputCategory, cfg.OnLabel, cfg.OffLabel, cfg.OkValue, cfg.IsBinary, cfg.Hidden,
	)
	if err != nil {
		return fmt.Errorf("init probe config %s: %w", cfg.ProbeName, err)
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
