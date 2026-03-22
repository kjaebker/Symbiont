package api

import (
	"net/http"
	"time"
)

type probeResponse struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Type        string  `json:"type"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	TS          string  `json:"ts"`
	Status      string  `json:"status"`
}

func (s *Server) HandleProbeList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	readings, err := s.duck.CurrentProbeReadings(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe readings", "db_error")
		return
	}

	// Load all probe configs for merging.
	configs, err := s.sqlite.ListProbeConfigs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe configs", "db_error")
		return
	}
	cfgMap := make(map[string]*probeConfigLookup, len(configs))
	for i := range configs {
		c := &configs[i]
		cfgMap[c.ProbeName] = &probeConfigLookup{
			displayName: derefStr(c.DisplayName, c.ProbeName),
			unit:        derefStr(c.UnitOverride, ""),
			minNormal:   c.MinNormal,
			maxNormal:   c.MaxNormal,
			minWarning:  c.MinWarning,
			maxWarning:  c.MaxWarning,
		}
	}

	probes := make([]probeResponse, 0, len(readings))
	for _, rd := range readings {
		cfg := cfgMap[rd.Name]
		displayName := rd.Name
		unit := probeTypeToUnit(rd.Type)
		status := "unknown"

		if cfg != nil {
			displayName = cfg.displayName
			if cfg.unit != "" {
				unit = cfg.unit
			}
			status = computeProbeStatus(rd.Value, cfg)
		}

		probes = append(probes, probeResponse{
			Name:        rd.Name,
			DisplayName: displayName,
			Type:        rd.Type,
			Value:       rd.Value,
			Unit:        unit,
			TS:          rd.Timestamp.Format(time.RFC3339),
			Status:      status,
		})
	}

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
	case d <= 2*time.Hour:
		return "10s"
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

type probeConfigLookup struct {
	displayName string
	unit        string
	minNormal   *float64
	maxNormal   *float64
	minWarning  *float64
	maxWarning  *float64
}

// computeProbeStatus determines probe status from config thresholds.
func computeProbeStatus(value float64, cfg *probeConfigLookup) string {
	if cfg == nil {
		return "unknown"
	}

	hasNormal := cfg.minNormal != nil || cfg.maxNormal != nil
	hasWarning := cfg.minWarning != nil || cfg.maxWarning != nil

	if !hasNormal && !hasWarning {
		return "unknown"
	}

	// Check critical (outside warning thresholds).
	if hasWarning {
		if cfg.minWarning != nil && value < *cfg.minWarning {
			return "critical"
		}
		if cfg.maxWarning != nil && value > *cfg.maxWarning {
			return "critical"
		}
	}

	// Check warning (outside normal thresholds but within warning).
	if hasNormal {
		if cfg.minNormal != nil && value < *cfg.minNormal {
			return "warning"
		}
		if cfg.maxNormal != nil && value > *cfg.maxNormal {
			return "warning"
		}
	}

	return "normal"
}

func probeTypeToUnit(probeType string) string {
	switch probeType {
	case "Temp":
		return "F"
	case "pH":
		return "pH"
	case "Amps":
		return "A"
	case "pwr":
		return "W"
	case "volts":
		return "V"
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

