package db

import (
	"context"
	"encoding/json"
	"fmt"
)

// --- Agent Settings ---

// GetAgentSettings returns the single row of agent_settings, inserting defaults
// on first access so callers can always rely on a non-nil result.
func (s *SQLiteDB) GetAgentSettings(ctx context.Context) (*AgentSettings, error) {
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO agent_settings (id) VALUES (1)`); err != nil {
		return nil, fmt.Errorf("seeding agent_settings: %w", err)
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT tone, dosing_product_line, net_volume_gallons,
		        custom_guardrails, enabled_skills, updated_at
		 FROM agent_settings WHERE id = 1`)
	a := &AgentSettings{}
	var enabledJSON string
	if err := row.Scan(&a.Tone, &a.DosingProductLine, &a.NetVolumeGallons,
		&a.CustomGuardrails, &enabledJSON, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("getting agent settings: %w", err)
	}
	if enabledJSON != "" {
		if err := json.Unmarshal([]byte(enabledJSON), &a.EnabledSkills); err != nil {
			return nil, fmt.Errorf("parsing enabled_skills: %w", err)
		}
	}
	if a.EnabledSkills == nil {
		a.EnabledSkills = []string{}
	}
	return a, nil
}

// UpsertAgentSettings writes the full agent_settings row.
func (s *SQLiteDB) UpsertAgentSettings(ctx context.Context, a AgentSettings) error {
	skills := a.EnabledSkills
	if skills == nil {
		skills = []string{}
	}
	enabledJSON, err := json.Marshal(skills)
	if err != nil {
		return fmt.Errorf("encoding enabled_skills: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO agent_settings
		    (id, tone, dosing_product_line, net_volume_gallons,
		     custom_guardrails, enabled_skills, updated_at)
		 VALUES (1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		    tone                = excluded.tone,
		    dosing_product_line = excluded.dosing_product_line,
		    net_volume_gallons  = excluded.net_volume_gallons,
		    custom_guardrails   = excluded.custom_guardrails,
		    enabled_skills      = excluded.enabled_skills,
		    updated_at          = CURRENT_TIMESTAMP`,
		a.Tone, a.DosingProductLine, a.NetVolumeGallons,
		a.CustomGuardrails, string(enabledJSON),
	)
	if err != nil {
		return fmt.Errorf("upserting agent settings: %w", err)
	}
	return nil
}
