package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/kjaebker/symbiont/internal/db"
)

func (s *Server) HandleAlertList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rules, err := s.sqlite.ListAlertRules(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch alert rules", "db_error")
		return
	}

	configs, err := s.sqlite.ListProbeConfigs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe configs", "db_error")
		return
	}
	cfgMap := make(map[string]*db.ProbeConfig, len(configs))
	for i := range configs {
		cfgMap[configs[i].ProbeName] = &configs[i]
	}

	// For probe_range rules, range_low/range_high are the probe's configured
	// bounds the rule currently follows, and range_configured says whether the
	// probe's config can trigger the rule at all.
	type ruleWithDisplay struct {
		db.AlertRule
		ProbeDisplayName string   `json:"probe_display_name"`
		RangeLow         *float64 `json:"range_low"`
		RangeHigh        *float64 `json:"range_high"`
		RangeConfigured  bool     `json:"range_configured"`
	}
	out := make([]ruleWithDisplay, len(rules))
	for i, r := range rules {
		cfg := cfgMap[r.ProbeName]
		dn := ""
		if cfg != nil {
			dn = derefStr(cfg.DisplayName, "")
		}
		if dn == "" {
			dn = splitCamelCase(r.ProbeName)
		}
		out[i] = ruleWithDisplay{AlertRule: r, ProbeDisplayName: dn}
		if r.Condition == db.AlertConditionProbeRange {
			out[i].RangeLow, out[i].RangeHigh = cfg.RangeFor(r.Severity)
			out[i].RangeConfigured = cfg.CanReach(r.Severity)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"rules": out})
}

// alertRuleInput accepts all fields that the list response returns so that
// round-tripped payloads (id, created_at, probe_display_name) aren't rejected
// by DisallowUnknownFields in readJSON. Read-only and computed fields are
// accepted but ignored when writing.
type alertRuleInput struct {
	ID               int64    `json:"id"`
	ProbeName        string   `json:"probe_name"`
	Condition        string   `json:"condition"`
	ThresholdLow     *float64 `json:"threshold_low"`
	ThresholdHigh    *float64 `json:"threshold_high"`
	Severity         string   `json:"severity"`
	CooldownMinutes  int      `json:"cooldown_minutes"`
	Enabled          bool     `json:"enabled"`
	CreatedAt        string   `json:"created_at"`
	ProbeDisplayName string   `json:"probe_display_name"`
	RangeLow         *float64 `json:"range_low"`
	RangeHigh        *float64 `json:"range_high"`
	RangeConfigured  bool     `json:"range_configured"`
}

// toRule converts request input to a rule. probe_range rules carry no
// thresholds of their own, so any sent with them are dropped.
func (in alertRuleInput) toRule() db.AlertRule {
	rule := db.AlertRule{
		ProbeName:       in.ProbeName,
		Condition:       in.Condition,
		ThresholdLow:    in.ThresholdLow,
		ThresholdHigh:   in.ThresholdHigh,
		Severity:        in.Severity,
		CooldownMinutes: in.CooldownMinutes,
		Enabled:         in.Enabled,
	}
	if rule.Condition == db.AlertConditionProbeRange {
		rule.ThresholdLow, rule.ThresholdHigh = nil, nil
	}
	return rule
}

// checkProbeRange rejects a probe_range rule on a probe whose configured
// ranges could never trigger it, returning an error message or "".
func (s *Server) checkProbeRange(r *http.Request, rule db.AlertRule) string {
	if rule.Condition != db.AlertConditionProbeRange {
		return ""
	}
	// A missing config row is the same as no ranges; any other error is real.
	cfg, err := s.sqlite.GetProbeConfig(r.Context(), rule.ProbeName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("failed to load probe config for alert rule", "err", err, "probe", rule.ProbeName)
		return "could not check this probe's configured range"
	}
	if !cfg.CanReach(rule.Severity) {
		if rule.Severity == db.ProbeStatusCritical {
			return "this probe has no critical limits configured; set them in Settings > Probes or use a custom threshold"
		}
		return "this probe has no normal range configured; set one in Settings > Probes or use a custom threshold"
	}
	return ""
}

func (s *Server) HandleAlertCreate(w http.ResponseWriter, r *http.Request) {
	var input alertRuleInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	rule := input.toRule()
	if err := validateAlertRule(rule); err != "" {
		writeError(w, http.StatusBadRequest, err, "validation_error")
		return
	}
	if err := s.checkProbeRange(r, rule); err != "" {
		writeError(w, http.StatusBadRequest, err, "range_not_configured")
		return
	}

	id, insertErr := s.sqlite.InsertAlertRule(r.Context(), rule)
	if insertErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to create alert rule", "db_error")
		return
	}

	rule.ID = id
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) HandleAlertUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid alert rule id", "invalid_param")
		return
	}

	var input alertRuleInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	rule := input.toRule()
	if errMsg := validateAlertRule(rule); errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg, "validation_error")
		return
	}
	if errMsg := s.checkProbeRange(r, rule); errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg, "range_not_configured")
		return
	}

	if err := s.sqlite.UpdateAlertRule(r.Context(), id, rule); err != nil {
		writeError(w, http.StatusNotFound, "alert rule not found", "not_found")
		return
	}

	rule.ID = id
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) HandleAlertDelete(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid alert rule id", "invalid_param")
		return
	}

	if err := s.sqlite.DeleteAlertRule(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "alert rule not found", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleAlertEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := queryParam(r, "limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}

	var ruleID *int64
	if idStr := r.URL.Query().Get("rule_id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			ruleID = &id
		}
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"

	events, err := s.sqlite.ListAlertEvents(r.Context(), ruleID, activeOnly, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch alert events", "db_error")
		return
	}

	displayNames := s.probeDisplayNames(r.Context())

	type eventWithDisplay struct {
		db.AlertEvent
		ProbeDisplayName string `json:"probe_display_name"`
	}
	out := make([]eventWithDisplay, len(events))
	for i, e := range events {
		dn := ""
		if e.ProbeName != nil {
			dn = displayNames[*e.ProbeName]
			if dn == "" {
				dn = splitCamelCase(*e.ProbeName)
			}
		}
		out[i] = eventWithDisplay{AlertEvent: e, ProbeDisplayName: dn}
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

// validateAlertRule returns an error message if the rule is invalid, or empty string if valid.
func validateAlertRule(rule db.AlertRule) string {
	if rule.ProbeName == "" {
		return "probe_name is required"
	}

	switch rule.Condition {
	case "above", "below", "outside_range", db.AlertConditionProbeRange:
		// valid
	default:
		return "condition must be 'probe_range', 'above', 'below', or 'outside_range'"
	}

	switch rule.Severity {
	case "warning", "critical":
		// valid
	default:
		return "severity must be 'warning' or 'critical'"
	}

	if rule.Condition == "above" && rule.ThresholdHigh == nil {
		return "threshold_high is required for 'above' condition"
	}
	if rule.Condition == "below" && rule.ThresholdLow == nil {
		return "threshold_low is required for 'below' condition"
	}
	if rule.Condition == "outside_range" && (rule.ThresholdLow == nil || rule.ThresholdHigh == nil) {
		return "threshold_low and threshold_high are required for 'outside_range' condition"
	}

	return ""
}
