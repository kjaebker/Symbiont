package api

import "net/http"

func (s *Server) HandleFeedGet(w http.ResponseWriter, r *http.Request) {
	status, err := s.apex.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch feed status", "apex_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":   status.Feed.Name,
		"active": status.Feed.Active,
	})
}

func (s *Server) HandleFeedSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   int  `json:"name"`
		Active bool `json:"active"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Name < 0 || body.Name > 4 {
		writeError(w, http.StatusBadRequest, "name must be 0–4 (0=cancel, 1–4=Feed A–D)", "invalid_param")
		return
	}
	if !body.Active {
		body.Name = 0
	}

	if err := s.apex.SetFeedMode(r.Context(), body.Name, body.Active); err != nil {
		writeError(w, http.StatusBadGateway, "failed to set feed mode: "+err.Error(), "apex_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":   body.Name,
		"active": body.Active,
	})
}
