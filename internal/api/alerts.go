package api

import (
	"net/http"
	"strconv"

	"github.com/kjaebker/symbiont/internal/db"
)

func (s *Server) HandleAlertList(w http.ResponseWriter, r *http.Request) {
	rules, err := s.sqlite.ListAlertRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch alert rules", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (s *Server) HandleAlertCreate(w http.ResponseWriter, r *http.Request) {
	var rule db.AlertRule
	if err := readJSON(r, &rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if err := validateAlertRule(rule); err != "" {
		writeError(w, http.StatusBadRequest, err, "validation_error")
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

	var rule db.AlertRule
	if err := readJSON(r, &rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if errMsg := validateAlertRule(rule); errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg, "validation_error")
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
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// validateAlertRule returns an error message if the rule is invalid, or empty string if valid.
func validateAlertRule(rule db.AlertRule) string {
	if rule.ProbeName == "" {
		return "probe_name is required"
	}

	switch rule.Condition {
	case "above", "below", "outside_range":
		// valid
	default:
		return "condition must be 'above', 'below', or 'outside_range'"
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
