package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// HandleKitList returns all test kit definitions grouped by parameter name.
func (s *Server) HandleKitList(w http.ResponseWriter, r *http.Request) {
	if s.catalog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"kits": map[string]any{}})
		return
	}

	type KitResponse struct {
		Ref        string  `json:"ref"`
		Name       string  `json:"name"`
		InputLabel string  `json:"input_label"`
		InputUnit  string  `json:"input_unit"`
		Scale      float64 `json:"scale"`
		Offset     float64 `json:"offset"`
	}

	grouped := s.catalog.GroupedByParameter()
	result := make(map[string][]KitResponse, len(grouped))
	for param, kits := range grouped {
		resp := make([]KitResponse, 0, len(kits))
		for _, k := range kits {
			resp = append(resp, KitResponse{
				Ref:        k.Ref,
				Name:       k.Name,
				InputLabel: k.Input.Label,
				InputUnit:  k.Input.Unit,
				Scale:      k.Conversion.Scale,
				Offset:     k.Conversion.Offset,
			})
		}
		result[param] = resp
	}
	writeJSON(w, http.StatusOK, map[string]any{"kits": result})
}

// HandleMeasurementParameterList returns all measurement parameter definitions.
func (s *Server) HandleMeasurementParameterList(w http.ResponseWriter, r *http.Request) {
	params, err := s.sqlite.ListMeasurementParameters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch parameters", "db_error")
		return
	}
	if params == nil {
		params = []db.MeasurementParameter{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"parameters": params})
}

// HandleMeasurementList returns measurements, optionally filtered by parameter, date range, and limit.
func (s *Server) HandleMeasurementList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	f := db.MeasurementFilter{}

	if paramName := r.URL.Query().Get("parameter"); paramName != "" {
		p, err := s.sqlite.GetMeasurementParameterByName(ctx, paramName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to lookup parameter", "db_error")
			return
		}
		if p == nil {
			writeError(w, http.StatusBadRequest, "unknown parameter: "+paramName, "invalid_parameter")
			return
		}
		f.ParameterID = &p.ID
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' timestamp (RFC3339 required)", "invalid_param")
			return
		}
		f.From = &t
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' timestamp (RFC3339 required)", "invalid_param")
			return
		}
		f.To = &t
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		n, err := strconv.Atoi(limitStr)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit", "invalid_param")
			return
		}
		if n > 500 {
			n = 500
		}
		f.Limit = n
	}

	measurements, err := s.sqlite.ListMeasurements(ctx, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch measurements", "db_error")
		return
	}
	if measurements == nil {
		measurements = []db.Measurement{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"measurements": measurements})
}

// HandleMeasurementCreate adds a new manual measurement entry.
func (s *Server) HandleMeasurementCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Parameter  string   `json:"parameter"`
		Value      float64  `json:"value"`
		MeasuredAt string   `json:"measured_at"` // RFC3339; defaults to now
		Notes      *string  `json:"notes"`
		TestKitRef *string  `json:"test_kit_ref"`
		RawValue   *float64 `json:"raw_value"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if req.Parameter == "" {
		writeError(w, http.StatusBadRequest, "parameter is required", "validation_error")
		return
	}

	param, err := s.sqlite.GetMeasurementParameterByName(ctx, req.Parameter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to lookup parameter", "db_error")
		return
	}
	if param == nil {
		writeError(w, http.StatusBadRequest, "unknown parameter: "+req.Parameter, "invalid_parameter")
		return
	}

	measuredAt := time.Now().UTC()
	if req.MeasuredAt != "" {
		t, err := time.Parse(time.RFC3339, req.MeasuredAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid measured_at (RFC3339 required)", "invalid_param")
			return
		}
		measuredAt = t.UTC()
	}

	// If a kit ref is provided, validate it exists in the catalog.
	if req.TestKitRef != nil && *req.TestKitRef != "" {
		if s.catalog != nil {
			if _, ok := s.catalog.Get(*req.TestKitRef); !ok {
				writeError(w, http.StatusBadRequest, "unknown test_kit_ref: "+*req.TestKitRef, "invalid_kit_ref")
				return
			}
		}
		// If raw_value is provided with a kit ref, compute canonical value.
		if req.RawValue != nil && s.catalog != nil {
			canonical, applyErr := s.catalog.Apply(*req.TestKitRef, *req.RawValue)
			if applyErr == nil {
				req.Value = canonical
			}
		}
	}

	m := db.Measurement{
		MeasuredAt:  measuredAt,
		ParameterID: param.ID,
		Value:       req.Value,
		Notes:       req.Notes,
		Source:      "manual",
		TestKitRef:  req.TestKitRef,
		RawValue:    req.RawValue,
	}

	id, err := s.sqlite.InsertMeasurement(ctx, m)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create measurement", "db_error")
		return
	}

	created, err := s.sqlite.GetMeasurement(ctx, id)
	if err != nil || created == nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created measurement", "db_error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// HandleMeasurementUpdate updates a measurement's value, date, notes, or kit ref.
func (s *Server) HandleMeasurementUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid measurement id", "invalid_param")
		return
	}

	existing, err := s.sqlite.GetMeasurement(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch measurement", "db_error")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "measurement not found", "not_found")
		return
	}

	var req struct {
		Parameter  *string  `json:"parameter"`
		Value      *float64 `json:"value"`
		MeasuredAt *string  `json:"measured_at"`
		Notes      *string  `json:"notes"`
		TestKitRef *string  `json:"test_kit_ref"`
		RawValue   *float64 `json:"raw_value"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	// Apply updates over existing values.
	updated := db.Measurement{
		MeasuredAt:  existing.MeasuredAt,
		ParameterID: existing.ParameterID,
		Value:       existing.Value,
		Notes:       existing.Notes,
		TestKitRef:  existing.TestKitRef,
		RawValue:    existing.RawValue,
	}

	if req.Parameter != nil {
		param, err := s.sqlite.GetMeasurementParameterByName(ctx, *req.Parameter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to lookup parameter", "db_error")
			return
		}
		if param == nil {
			writeError(w, http.StatusBadRequest, "unknown parameter: "+*req.Parameter, "invalid_parameter")
			return
		}
		updated.ParameterID = param.ID
	}
	if req.Value != nil {
		updated.Value = *req.Value
	}
	if req.MeasuredAt != nil {
		t, err := time.Parse(time.RFC3339, *req.MeasuredAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid measured_at (RFC3339 required)", "invalid_param")
			return
		}
		updated.MeasuredAt = t.UTC()
	}
	if req.Notes != nil {
		updated.Notes = req.Notes
	}
	if req.TestKitRef != nil {
		updated.TestKitRef = req.TestKitRef
	}
	if req.RawValue != nil {
		updated.RawValue = req.RawValue
	}

	if err := s.sqlite.UpdateMeasurement(ctx, id, updated); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update measurement", "db_error")
		return
	}

	result, err := s.sqlite.GetMeasurement(ctx, id)
	if err != nil || result == nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated measurement", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleMeasurementDelete removes a measurement by ID.
func (s *Server) HandleMeasurementDelete(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid measurement id", "invalid_param")
		return
	}

	if err := s.sqlite.DeleteMeasurement(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "measurement not found", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
