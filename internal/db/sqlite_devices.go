package db

import (
	"context"
	"database/sql"
	"fmt"
)

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

	// Load probe names and visualization outlets for each device.
	for i := range devices {
		probes, err := s.listDeviceProbeNames(ctx, devices[i].ID)
		if err != nil {
			return nil, err
		}
		devices[i].ProbeNames = probes

		outlets, err := s.ListDeviceOutlets(ctx, devices[i].ID)
		if err != nil {
			return nil, err
		}
		devices[i].OutletIDs = outlets
	}
	return devices, nil
}

// GetDevice returns a single device by ID with its linked probe names and visualization outlets.
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

	outlets, err := s.ListDeviceOutlets(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.OutletIDs = outlets
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

// ListDeviceOutlets returns the visualization outlets linked to a device.
func (s *SQLiteDB) ListDeviceOutlets(ctx context.Context, deviceID int64) ([]DeviceOutlet, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT device_id, outlet_id, label, color, sort_order
		FROM device_outlets WHERE device_id = ? ORDER BY sort_order, outlet_id`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing device outlets for device %d: %w", deviceID, err)
	}
	defer rows.Close()

	var outlets []DeviceOutlet
	for rows.Next() {
		var o DeviceOutlet
		if err := rows.Scan(&o.DeviceID, &o.OutletID, &o.Label, &o.Color, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("scanning device outlet: %w", err)
		}
		outlets = append(outlets, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device outlets: %w", err)
	}
	if outlets == nil {
		outlets = []DeviceOutlet{}
	}
	return outlets, nil
}

// SetDeviceOutlets replaces the full set of visualization outlets for a device.
func (s *SQLiteDB) SetDeviceOutlets(ctx context.Context, deviceID int64, outlets []DeviceOutlet) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM device_outlets WHERE device_id = ?", deviceID); err != nil {
		return fmt.Errorf("clearing device outlets for device %d: %w", deviceID, err)
	}

	for _, o := range outlets {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO device_outlets (device_id, outlet_id, label, color, sort_order) VALUES (?, ?, ?, ?, ?)`,
			deviceID, o.OutletID, o.Label, o.Color, o.SortOrder,
		); err != nil {
			return fmt.Errorf("inserting device outlet %s: %w", o.OutletID, err)
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

	names := make([]string, 0)
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scanning probe name: %w", err)
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
