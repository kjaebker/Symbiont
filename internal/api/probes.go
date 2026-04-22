package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
)

type probeResponse struct {
	Name          string  `json:"name"`
	DisplayName   string  `json:"display_name"`
	Type          string  `json:"type"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	TS            string  `json:"ts"`
	Status        string  `json:"status"`
	InputCategory string  `json:"input_category"`
	OnLabel       *string `json:"on_label"`
	OffLabel      *string `json:"off_label"`
	IsBinary      bool    `json:"is_binary"`
	Hidden        bool    `json:"hidden"`
}

func (s *Server) HandleProbeList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	readings, err := s.duck.CurrentProbeReadings(ctx)
	if err != nil {
		s.logger.Error("failed to fetch probe readings", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch probe readings", "db_error")
		return
	}

	// Load all probe configs for merging.
	configs, err := s.sqlite.ListProbeConfigs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe configs", "db_error")
		return
	}
	cfgMap := make(map[string]*db.ProbeConfig, len(configs))
	for i := range configs {
		cfgMap[configs[i].ProbeName] = &configs[i]
	}

	// Auto-init classification defaults for any probe seen for the first time.
	for _, rd := range readings {
		if _, exists := cfgMap[rd.Name]; !exists {
			cls := apex.ClassifyInput(apex.Input{Name: rd.Name, Type: rd.Type})
			_ = s.sqlite.InitProbeConfig(ctx, db.ProbeConfig{
				ProbeName:     rd.Name,
				InputCategory: cls.Category,
				OnLabel:       cls.OnLabel,
				OffLabel:      cls.OffLabel,
				OkValue:       cls.OkValue,
				IsBinary:      cls.IsBinary,
				Hidden:        cls.Hidden,
			})
		}
	}

	probes := make([]probeResponse, 0, len(readings))
	for _, rd := range readings {
		c := cfgMap[rd.Name]

		displayName := splitCamelCase(rd.Name)
		unit := probeTypeToUnit(rd.Type)
		category := "probe"
		var onLabel, offLabel *string
		isBinary := false
		hidden := false

		if c != nil {
			displayName = derefStr(c.DisplayName, displayName)
			if c.UnitOverride != nil && *c.UnitOverride != "" {
				unit = *c.UnitOverride
			}
			category = c.InputCategory
			onLabel = c.OnLabel
			offLabel = c.OffLabel
			isBinary = c.IsBinary
			hidden = c.Hidden
		}

		status := computeProbeStatus(rd.Value, c)

		probes = append(probes, probeResponse{
			Name:          rd.Name,
			DisplayName:   displayName,
			Type:          rd.Type,
			Value:         rd.Value,
			Unit:          unit,
			TS:            rd.Timestamp.Format(time.RFC3339),
			Status:        status,
			InputCategory: category,
			OnLabel:       onLabel,
			OffLabel:      offLabel,
			IsBinary:      isBinary,
			Hidden:        hidden,
		})
	}

	// Sort alphabetically by name.
	sort.SliceStable(probes, func(i, j int) bool {
		return probes[i].Name < probes[j].Name
	})

	var polledAt string
	if len(readings) > 0 {
		polledAt = readings[0].Timestamp.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"probes":    probes,
		"polled_at": polledAt,
	})
}

func (s *Server) HandleProbeHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := pathValue(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "probe name is required", "missing_param")
		return
	}

	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' timestamp", "invalid_param")
			return
		}
		from = t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' timestamp", "invalid_param")
			return
		}
		to = t
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = autoInterval(to.Sub(from))
	}
	if !validInterval(interval) {
		writeError(w, http.StatusBadRequest, "invalid interval value", "invalid_param")
		return
	}

	data, err := s.duck.ProbeHistory(ctx, name, from, to, interval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe history", "db_error")
		return
	}

	if data == nil {
		writeError(w, http.StatusNotFound, "probe not found or no data in range", "not_found")
		return
	}

	type dataPoint struct {
		TS    string  `json:"ts"`
		Value float64 `json:"value"`
	}
	points := make([]dataPoint, 0, len(data))
	for _, dp := range data {
		points = append(points, dataPoint{
			TS:    dp.Timestamp.Format(time.RFC3339),
			Value: dp.Value,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"probe":    name,
		"from":     from.Format(time.RFC3339),
		"to":       to.Format(time.RFC3339),
		"interval": interval,
		"data":     points,
	})
}

// autoInterval selects a bucket interval based on the query time range.
func autoInterval(d time.Duration) string {
	switch {
	case d <= 12*time.Hour:
		return "1m"
	case d <= 3*24*time.Hour:
		return "5m"
	case d <= 14*24*time.Hour:
		return "15m"
	case d <= 60*24*time.Hour:
		return "1h"
	default:
		return "1d"
	}
}

var validIntervals = map[string]bool{
	"10s": true, "30s": true, "1m": true, "5m": true,
	"15m": true, "1h": true, "1d": true,
}

// validInterval checks if the given interval is in the allowlist.
func validInterval(interval string) bool {
	return validIntervals[interval]
}

// computeProbeStatus determines probe status from config thresholds.
// For binary categories (fluid/alarm) with ok_value set, status is derived
// from whether the current value matches the expected "OK" value.
// For analog probes, standard min/max threshold logic applies.
func computeProbeStatus(value float64, cfg *db.ProbeConfig) string {
	if cfg == nil {
		return "unknown"
	}

	// Binary inputs: derive status from ok_value if set.
	if cfg.IsBinary && cfg.OkValue != nil {
		if value == *cfg.OkValue {
			return "normal"
		}
		switch cfg.InputCategory {
		case "alarm":
			return "critical"
		case "fluid":
			return "warning"
		}
		return "unknown"
	}

	// Analog probes: threshold-based status.
	hasNormal := cfg.MinNormal != nil || cfg.MaxNormal != nil
	hasWarning := cfg.MinWarning != nil || cfg.MaxWarning != nil

	if !hasNormal && !hasWarning {
		return "unknown"
	}

	if hasWarning {
		if cfg.MinWarning != nil && value < *cfg.MinWarning {
			return "critical"
		}
		if cfg.MaxWarning != nil && value > *cfg.MaxWarning {
			return "critical"
		}
	}

	if hasNormal {
		if cfg.MinNormal != nil && value < *cfg.MinNormal {
			return "warning"
		}
		if cfg.MaxNormal != nil && value > *cfg.MaxNormal {
			return "warning"
		}
	}

	return "normal"
}

func probeTypeToUnit(probeType string) string {
	switch probeType {
	case "Temp":
		return "°F"
	case "pH":
		return "pH"
	case "Amps":
		return "Amps"
	case "pwr":
		return "Watts"
	case "volts":
		return "Volts"
	default:
		return ""
	}
}

func derefStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}


