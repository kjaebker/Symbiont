package api

import (
	"net/http"

	"github.com/kjaebker/symbiont/internal/db"
)

func (s *Server) HandleProbeConfigList(w http.ResponseWriter, r *http.Request) {
	configs, err := s.sqlite.ListProbeConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probe configs", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configs": configs})
}

func (s *Server) HandleProbeConfigUpdate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "probe name is required", "missing_param")
		return
	}

	var cfg db.ProbeConfig
	if err := readJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	cfg.ProbeName = name

	if err := s.sqlite.UpsertProbeConfig(r.Context(), cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update probe config", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) HandleOutletConfigList(w http.ResponseWriter, r *http.Request) {
	configs, err := s.sqlite.ListOutletConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlet configs", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configs": configs})
}

func (s *Server) HandleOutletConfigUpdate(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "outlet id is required", "missing_param")
		return
	}

	var cfg db.OutletConfig
	if err := readJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	cfg.OutletID = id

	if err := s.sqlite.UpsertOutletConfig(r.Context(), cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update outlet config", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}
