package api

import (
	"net/http"
	"strconv"
)

func (s *Server) HandleTokenList(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.sqlite.ListTokens(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tokens", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

func (s *Server) HandleTokenCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label string `json:"label"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required", "missing_param")
		return
	}

	token, err := s.sqlite.InsertToken(r.Context(), body.Label)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token", "db_error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"token": token,
		"label": body.Label,
	})
}

func (s *Server) HandleTokenDelete(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id", "invalid_param")
		return
	}

	if err := s.sqlite.DeleteToken(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "token not found", "not_found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
