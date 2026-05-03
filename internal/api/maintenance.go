package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)

// HandleMaintenanceTaskList returns all maintenance tasks.
func (s *Server) HandleMaintenanceTaskList(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.sqlite.ListMaintenanceTasks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tasks", "db_error")
		return
	}
	if tasks == nil {
		tasks = []db.MaintenanceTask{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// HandleMaintenanceTaskCreate creates a new maintenance task.
func (s *Server) HandleMaintenanceTaskCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string   `json:"name"`
		Description  *string  `json:"description"`
		Frequency    string   `json:"frequency"`
		IntervalDays *float64 `json:"interval_days"`
		DayOfWeek    *int     `json:"day_of_week"`
		Enabled      *bool    `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if body.Name == "" || body.Frequency == "" {
		writeError(w, http.StatusBadRequest, "name and frequency are required", "validation_error")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	id, err := s.sqlite.InsertMaintenanceTask(r.Context(), db.MaintenanceTask{
		Name:         body.Name,
		Description:  body.Description,
		Frequency:    body.Frequency,
		IntervalDays: body.IntervalDays,
		DayOfWeek:    body.DayOfWeek,
		Enabled:      enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task", "db_error")
		return
	}
	task, err := s.sqlite.GetMaintenanceTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created task", "db_error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"task": task})
}

// HandleMaintenanceTaskUpdate updates a maintenance task.
func (s *Server) HandleMaintenanceTaskUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id", "bad_request")
		return
	}
	var body struct {
		Name         string   `json:"name"`
		Description  *string  `json:"description"`
		Frequency    string   `json:"frequency"`
		IntervalDays *float64 `json:"interval_days"`
		DayOfWeek    *int     `json:"day_of_week"`
		Enabled      bool     `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if err := s.sqlite.UpdateMaintenanceTask(r.Context(), db.MaintenanceTask{
		ID: id, Name: body.Name, Description: body.Description,
		Frequency: body.Frequency, IntervalDays: body.IntervalDays,
		DayOfWeek: body.DayOfWeek, Enabled: body.Enabled,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task", "db_error")
		return
	}
	task, err := s.sqlite.GetMaintenanceTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated task", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task})
}

// HandleMaintenanceTaskDelete deletes a maintenance task.
func (s *Server) HandleMaintenanceTaskDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id", "bad_request")
		return
	}
	if err := s.sqlite.DeleteMaintenanceTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleMaintenanceTaskComplete records a task completion.
func (s *Server) HandleMaintenanceTaskComplete(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id", "bad_request")
		return
	}

	var body struct {
		CompletedAt *string `json:"completed_at"`
		Notes       *string `json:"notes"`
		Source      string  `json:"source"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	task, err := s.sqlite.GetMaintenanceTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found", "not_found")
		return
	}

	completedAt := time.Now()
	if body.CompletedAt != nil {
		t, err := time.Parse(time.RFC3339, *body.CompletedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid completed_at format (RFC3339 required)", "bad_request")
			return
		}
		completedAt = t
	}

	source := body.Source
	if source == "" {
		source = "manual"
	}

	logID, err := s.sqlite.InsertMaintenanceLog(r.Context(), db.MaintenanceLog{
		TaskID:      taskID,
		CompletedAt: completedAt,
		Notes:       body.Notes,
		Source:      source,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to log task completion", "db_error")
		return
	}

	s.events.Publish(events.NewMaintenanceCompleted(logID, taskID, task.Name, source, body.Notes))

	writeJSON(w, http.StatusCreated, map[string]any{"log_id": logID})
}

// HandleMaintenanceLogList returns log history for a task.
func (s *Server) HandleMaintenanceLogList(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id", "bad_request")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			limit = l
		}
	}
	logs, err := s.sqlite.ListMaintenanceLogs(r.Context(), taskID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch logs", "db_error")
		return
	}
	if logs == nil {
		logs = []db.MaintenanceLog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

// HandleDueItems returns all due and overdue dosing/maintenance items.
// Horizon defaults to now+24h (shows due today and overdue).
func (s *Server) HandleDueItems(w http.ResponseWriter, r *http.Request) {
	horizon := time.Now().Add(24 * time.Hour)
	if v := r.URL.Query().Get("horizon"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			horizon = t
		}
	}
	items, err := s.sqlite.ListDueItems(r.Context(), horizon)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch due items", "db_error")
		return
	}
	if items == nil {
		items = []db.DueItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
