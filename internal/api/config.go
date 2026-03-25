package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kjaebker/symbiont/internal/db"
)

func (s *Server) HandleProbeConfigList(w http.ResponseWriter, r *http.Request) {
	configs, err := s.sqlite.ListProbeConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe configs", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configs": configs})
}

func (s *Server) HandleProbeConfigUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := pathValue(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "probe name is required", "missing_param")
		return
	}

	// Decode into a generic map so we know which fields were actually sent.
	var patch map[string]json.RawMessage
	if err := readJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	// Load existing config (or start from defaults).
	existing, err := s.sqlite.GetProbeConfig(ctx, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to fetch existing config", "db_error")
		return
	}
	cfg := db.ProbeConfig{ProbeName: name}
	if existing != nil {
		cfg = *existing
	}

	// If display_name is being changed and the probe is linked to a device, reject.
	if _, ok := patch["display_name"]; ok {
		device, devErr := s.sqlite.GetDeviceByProbeName(ctx, name)
		if devErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to check device link", "db_error")
			return
		}
		if device != nil {
			writeError(w, http.StatusConflict,
				"display name is managed by device '"+device.Name+"'",
				"device_managed")
			return
		}
	}

	// Merge only the provided fields.
	if v, ok := patch["display_name"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			cfg.DisplayName = &s
		}
	}
	if v, ok := patch["unit_override"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			cfg.UnitOverride = &s
		}
	}
	if v, ok := patch["min_normal"]; ok {
		cfg.MinNormal = unmarshalOptionalFloat(v)
	}
	if v, ok := patch["max_normal"]; ok {
		cfg.MaxNormal = unmarshalOptionalFloat(v)
	}
	if v, ok := patch["min_warning"]; ok {
		cfg.MinWarning = unmarshalOptionalFloat(v)
	}
	if v, ok := patch["max_warning"]; ok {
		cfg.MaxWarning = unmarshalOptionalFloat(v)
	}

	if err := s.sqlite.UpsertProbeConfig(ctx, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update probe config", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// unmarshalOptionalFloat handles both null and numeric JSON values for nullable float fields.
func unmarshalOptionalFloat(raw json.RawMessage) *float64 {
	if string(raw) == "null" {
		return nil
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return &f
	}
	return nil
}

func (s *Server) HandleOutletConfigList(w http.ResponseWriter, r *http.Request) {
	configs, err := s.sqlite.ListOutletConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlet configs", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configs": configs})
}

func (s *Server) HandleOutletConfigUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := pathValue(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "outlet id is required", "missing_param")
		return
	}

	var patch map[string]json.RawMessage
	if err := readJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	existing, err := s.sqlite.GetOutletConfig(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to fetch existing config", "db_error")
		return
	}
	cfg := db.OutletConfig{OutletID: id}
	if existing != nil {
		cfg = *existing
	}

	// If display_name is being changed and the outlet is linked to a device, reject.
	if _, ok := patch["display_name"]; ok {
		device, devErr := s.sqlite.GetDeviceByOutletID(ctx, id)
		if devErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to check device link", "db_error")
			return
		}
		if device != nil {
			writeError(w, http.StatusConflict,
				"display name is managed by device '"+device.Name+"'",
				"device_managed")
			return
		}
	}

	if v, ok := patch["display_name"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			cfg.DisplayName = &s
		}
	}
	if v, ok := patch["icon"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			cfg.Icon = &s
		}
	}
	if err := s.sqlite.UpsertOutletConfig(ctx, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update outlet config", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}
