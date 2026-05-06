package api

import (
	"net/http"
	"strconv"

	"github.com/kjaebker/symbiont/internal/enums"
	"github.com/kjaebker/symbiont/internal/events"
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
		Scope string `json:"scope"` // read | control | admin (default: admin)
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required", "missing_param")
		return
	}
	scope := body.Scope
	if scope == "" {
		scope = "admin"
	}
	if !enums.TokenScopes.Has(scope) {
		writeError(w, http.StatusBadRequest, "invalid scope; must be: read, control, admin", "invalid_param")
		return
	}

	token, err := s.sqlite.InsertTokenWithScope(r.Context(), body.Label, scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token", "db_error")
		return
	}

	s.events.Publish(events.NewAuthEvent("token_created", body.Label))

	writeJSON(w, http.StatusCreated, map[string]string{
		"token": token,
		"label": body.Label,
		"scope": scope,
	})
}

func (s *Server) HandleTokenUpdateScope(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id", "invalid_param")
		return
	}

	var body struct {
		Scope string `json:"scope"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if !enums.TokenScopes.Has(body.Scope) {
		writeError(w, http.StatusBadRequest, "invalid scope; must be: read, control, admin", "invalid_param")
		return
	}

	if err := s.sqlite.UpdateTokenScope(r.Context(), id, body.Scope); err != nil {
		writeError(w, http.StatusNotFound, "token not found", "not_found")
		return
	}

	s.events.Publish(events.NewAuthEvent("token_scope_updated", idStr))

	writeJSON(w, http.StatusOK, map[string]string{"id": idStr, "scope": body.Scope})
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

	s.events.Publish(events.NewAuthEvent("token_deleted", idStr))

	w.WriteHeader(http.StatusNoContent)
}
