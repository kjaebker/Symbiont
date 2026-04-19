package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)

// HandleDosingProductList returns all dosing products.
func (s *Server) HandleDosingProductList(w http.ResponseWriter, r *http.Request) {
	products, err := s.sqlite.ListDosingProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch products", "db_error")
		return
	}
	if products == nil {
		products = []db.DosingProduct{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

// HandleDosingProductCreate creates a new dosing product.
func (s *Server) HandleDosingProductCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Brand string  `json:"brand"`
		Name  string  `json:"name"`
		Type  string  `json:"type"`
		Unit  string  `json:"unit"`
		Notes *string `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if body.Brand == "" || body.Name == "" || body.Type == "" {
		writeError(w, http.StatusBadRequest, "brand, name, and type are required", "validation_error")
		return
	}
	unit := body.Unit
	if unit == "" {
		unit = "mL"
	}
	id, err := s.sqlite.InsertDosingProduct(r.Context(), db.DosingProduct{
		Brand: body.Brand,
		Name:  body.Name,
		Type:  body.Type,
		Unit:  unit,
		Notes: body.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create product", "db_error")
		return
	}
	p, err := s.sqlite.GetDosingProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created product", "db_error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"product": p})
}

// HandleDosingProductUpdate updates a dosing product.
func (s *Server) HandleDosingProductUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id", "bad_request")
		return
	}
	var body struct {
		Brand string  `json:"brand"`
		Name  string  `json:"name"`
		Type  string  `json:"type"`
		Unit  string  `json:"unit"`
		Notes *string `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if err := s.sqlite.UpdateDosingProduct(r.Context(), db.DosingProduct{
		ID: id, Brand: body.Brand, Name: body.Name,
		Type: body.Type, Unit: body.Unit, Notes: body.Notes,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update product", "db_error")
		return
	}
	p, err := s.sqlite.GetDosingProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated product", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

// HandleDosingProductDelete deletes a dosing product.
func (s *Server) HandleDosingProductDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id", "bad_request")
		return
	}
	if err := s.sqlite.DeleteDosingProduct(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete product", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleDosingScheduleList returns all dosing schedules joined with product info.
func (s *Server) HandleDosingScheduleList(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.sqlite.ListDosingSchedules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch schedules", "db_error")
		return
	}
	if schedules == nil {
		schedules = []db.DosingSchedule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": schedules})
}

// HandleDosingScheduleCreate creates a new dosing schedule.
func (s *Server) HandleDosingScheduleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProductID           int64    `json:"product_id"`
		Amount              float64  `json:"amount"`
		Frequency           string   `json:"frequency"`
		IntervalDays        *float64 `json:"interval_days"`
		DayOfWeek           *int     `json:"day_of_week"`
		Enabled             *bool    `json:"enabled"`
		FollowUpParameterID *int64   `json:"follow_up_parameter_id"`
		FollowUpDays        *int     `json:"follow_up_days"`
		Notes               *string  `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if body.ProductID == 0 || body.Amount <= 0 || body.Frequency == "" {
		writeError(w, http.StatusBadRequest, "product_id, amount, and frequency are required", "validation_error")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	id, err := s.sqlite.InsertDosingSchedule(r.Context(), db.DosingSchedule{
		ProductID:           body.ProductID,
		Amount:              body.Amount,
		Frequency:           body.Frequency,
		IntervalDays:        body.IntervalDays,
		DayOfWeek:           body.DayOfWeek,
		Enabled:             enabled,
		FollowUpParameterID: body.FollowUpParameterID,
		FollowUpDays:        body.FollowUpDays,
		Notes:               body.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create schedule", "db_error")
		return
	}
	sched, err := s.sqlite.GetDosingSchedule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created schedule", "db_error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"schedule": sched})
}

// HandleDosingScheduleUpdate updates a dosing schedule.
func (s *Server) HandleDosingScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id", "bad_request")
		return
	}
	var body struct {
		ProductID           int64    `json:"product_id"`
		Amount              float64  `json:"amount"`
		Frequency           string   `json:"frequency"`
		IntervalDays        *float64 `json:"interval_days"`
		DayOfWeek           *int     `json:"day_of_week"`
		Enabled             bool     `json:"enabled"`
		FollowUpParameterID *int64   `json:"follow_up_parameter_id"`
		FollowUpDays        *int     `json:"follow_up_days"`
		Notes               *string  `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if err := s.sqlite.UpdateDosingSchedule(r.Context(), db.DosingSchedule{
		ID: id, ProductID: body.ProductID, Amount: body.Amount,
		Frequency: body.Frequency, IntervalDays: body.IntervalDays,
		DayOfWeek: body.DayOfWeek, Enabled: body.Enabled,
		FollowUpParameterID: body.FollowUpParameterID,
		FollowUpDays:        body.FollowUpDays,
		Notes:               body.Notes,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update schedule", "db_error")
		return
	}
	sched, err := s.sqlite.GetDosingSchedule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated schedule", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedule": sched})
}

// HandleDosingScheduleDelete deletes a dosing schedule.
func (s *Server) HandleDosingScheduleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id", "bad_request")
		return
	}
	if err := s.sqlite.DeleteDosingSchedule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete schedule", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleDosingScheduleLog records a dose for a schedule.
func (s *Server) HandleDosingScheduleLog(w http.ResponseWriter, r *http.Request) {
	schedID, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id", "bad_request")
		return
	}

	var body struct {
		Amount  *float64 `json:"amount"`
		DosedAt *string  `json:"dosed_at"`
		Notes   *string  `json:"notes"`
		Source  string   `json:"source"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	sched, err := s.sqlite.GetDosingSchedule(r.Context(), schedID)
	if err != nil {
		writeError(w, http.StatusNotFound, "schedule not found", "not_found")
		return
	}

	amount := sched.Amount
	if body.Amount != nil {
		amount = *body.Amount
	}

	dosedAt := time.Now()
	if body.DosedAt != nil {
		t, err := time.Parse(time.RFC3339, *body.DosedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dosed_at format (RFC3339 required)", "bad_request")
			return
		}
		dosedAt = t
	}

	source := body.Source
	if source == "" {
		source = "manual"
	}

	logID, err := s.sqlite.InsertDosingLog(r.Context(), db.DosingLog{
		ScheduleID: &schedID,
		ProductID:  sched.ProductID,
		Amount:     amount,
		DosedAt:    dosedAt,
		Notes:      body.Notes,
		Source:     source,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to log dose", "db_error")
		return
	}

	brand, name, unit := "", "", ""
	if sched.ProductBrand != nil {
		brand = *sched.ProductBrand
	}
	if sched.ProductName != nil {
		name = *sched.ProductName
	}
	if sched.ProductUnit != nil {
		unit = *sched.ProductUnit
	}
	s.events.Publish(events.NewDoseLogged(logID, sched.ProductID, &schedID, brand, name, amount, unit, source))

	writeJSON(w, http.StatusCreated, map[string]any{"log_id": logID})
}

// HandleDosingLogList returns dose log history.
func (s *Server) HandleDosingLogList(w http.ResponseWriter, r *http.Request) {
	f := db.DosingLogFilter{}

	if v := r.URL.Query().Get("product_id"); v != "" {
		pid, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product_id", "bad_request")
			return
		}
		f.ProductID = &pid
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err == nil {
			f.Limit = l
		}
	}

	logs, err := s.sqlite.ListDosingLogs(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch dose logs", "db_error")
		return
	}
	if logs == nil {
		logs = []db.DosingLog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}
