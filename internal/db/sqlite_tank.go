package db

import (
	"context"
	"database/sql"
	"fmt"
)

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
	// Normalize datetime values to date-only (YYYY-MM-DD).
	if p.SetupDate != nil && len(*p.SetupDate) > 10 {
		d := (*p.SetupDate)[:10]
		p.SetupDate = &d
	}
	return p, nil
}

// UpsertTankProfile inserts or replaces the tank profile for the given section.
func (s *SQLiteDB) UpsertTankProfile(ctx context.Context, p TankProfile) error {
	if p.SetupDate != nil && len(*p.SetupDate) > 10 {
		d := (*p.SetupDate)[:10]
		p.SetupDate = &d
	}
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
