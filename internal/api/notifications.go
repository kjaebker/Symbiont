package api

import (
	"net/http"
	"strconv"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/notify"
)

func (s *Server) HandleNotificationTargetList(w http.ResponseWriter, r *http.Request) {
	targets, err := s.sqlite.ListNotificationTargets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch notification targets", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"targets": targets})
}

func (s *Server) HandleNotificationTargetUpsert(w http.ResponseWriter, r *http.Request) {
	var target db.NotificationTarget
	if err := readJSON(r, &target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if target.Type == "" || target.Label == "" || target.Config == "" {
		writeError(w, http.StatusBadRequest, "type, label, and config are required", "validation_error")
		return
	}

	id, err := s.sqlite.UpsertNotificationTarget(r.Context(), target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save notification target", "db_error")
		return
	}

	target.ID = id
	writeJSON(w, http.StatusOK, target)
}

func (s *Server) HandleNotificationTargetDelete(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target id", "invalid_param")
		return
	}

	if err := s.sqlite.DeleteNotificationTarget(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "notification target not found", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleNotificationTest(w http.ResponseWriter, r *http.Request) {
	targets, err := s.sqlite.ListEnabledNotificationTargets(r.Context(), "ntfy")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load notification targets", "db_error")
		return
	}

	if len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "no enabled notification targets configured", "no_targets")
		return
	}

	type result struct {
		Label   string `json:"label"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	var results []result
	for _, t := range targets {
		n := notify.NewNtfy(t.Config)
		err := n.Send(r.Context(), notify.Notification{
			Title:    "Symbiont Test Notification",
			Body:     "If you see this, notifications are working correctly.",
			Priority: "default",
			Tags:     []string{"white_check_mark", "test"},
		})

		res := result{Label: t.Label, Success: err == nil}
		if err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
