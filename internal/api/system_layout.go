package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) HandleSystemLayoutGet(w http.ResponseWriter, r *http.Request) {
	sl, err := s.sqlite.GetSystemLayout(r.Context(), "default")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch system layout", "db_error")
		return
	}
	if sl == nil {
		writeJSON(w, http.StatusOK, map[string]any{"layout": nil})
		return
	}
	// Return layout as a raw JSON value, not a double-encoded string.
	writeJSON(w, http.StatusOK, map[string]any{"layout": json.RawMessage(sl.Layout)})
}

func (s *Server) HandleSystemLayoutSave(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Layout json.RawMessage `json:"layout"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Layout == nil {
		writeError(w, http.StatusBadRequest, "layout is required", "invalid_body")
		return
	}
	// Validate that layout is valid JSON.
	if !json.Valid(body.Layout) {
		writeError(w, http.StatusBadRequest, "layout must be valid JSON", "invalid_json")
		return
	}

	if err := s.sqlite.UpsertSystemLayout(r.Context(), "default", string(body.Layout)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save system layout", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
