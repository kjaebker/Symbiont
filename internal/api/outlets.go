package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
)

type outletResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	State       string  `json:"state"`
	Type        string  `json:"type"`
	Intensity   int     `json:"intensity"`
}

func (s *Server) HandleOutletList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	outlets, err := s.duck.CurrentOutletStates(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlet states", "db_error")
		return
	}

	// Load outlet configs for display name merging.
	configs, err := s.sqlite.ListOutletConfigs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlet configs", "db_error")
		return
	}
	cfgMap := make(map[string]string, len(configs))
	for _, c := range configs {
		if c.DisplayName != nil {
			cfgMap[c.OutletID] = *c.DisplayName
		}
	}

	resp := make([]outletResponse, 0, len(outlets))
	for _, o := range outlets {
		displayName := o.Name
		if dn, ok := cfgMap[o.DID]; ok {
			displayName = dn
		}
		resp = append(resp, outletResponse{
			ID:          o.DID,
			Name:        o.Name,
			DisplayName: displayName,
			State:       o.State,
			Type:        o.Type,
			Intensity:   o.Intensity,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"outlets": resp})
}

func (s *Server) HandleOutletSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	did := pathValue(r, "id")
	if did == "" {
		writeError(w, http.StatusBadRequest, "outlet id is required", "missing_param")
		return
	}

	var body struct {
		State string `json:"state"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	// Validate state.
	switch body.State {
	case "ON", "OFF", "AUTO":
		// valid
	default:
		writeError(w, http.StatusBadRequest, "state must be ON, OFF, or AUTO", "invalid_state")
		return
	}

	// Get current state for event log.
	outlets, err := s.duck.CurrentOutletStates(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch current outlet state", "db_error")
		return
	}
	var fromState *string
	var outletName *string
	for _, o := range outlets {
		if o.DID == did {
			fs := o.State
			fromState = &fs
			n := o.Name
			outletName = &n
			break
		}
	}

	// Send command to Apex.
	switch body.State {
	case "ON":
		err = s.apex.SetOutlet(ctx, did, apex.OutletOn)
	case "OFF":
		err = s.apex.SetOutlet(ctx, did, apex.OutletOff)
	case "AUTO":
		// The REST API doesn't support AUTO. Use the legacy CGI endpoint
		// which accepts state=0 to return an outlet to program control.
		if outletName == nil {
			writeError(w, http.StatusNotFound, "outlet not found", "not_found")
			return
		}
		err = s.apex.SetOutletAuto(ctx, *outletName)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to set outlet on apex: "+err.Error(), "apex_error")
		return
	}

	// Log the event.
	event := db.OutletEvent{
		OutletID:    did,
		OutletName:  outletName,
		FromState:   fromState,
		ToState:     body.State,
		InitiatedBy: "api",
	}
	if err := s.sqlite.InsertOutletEvent(ctx, event); err != nil {
		// Log but don't fail the request — the Apex command already succeeded.
		s.logger.Error("failed to log outlet event", "err", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":        did,
		"name":      derefStr(outletName, did),
		"state":     body.State,
		"logged_at": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) HandleOutletEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	outletID := r.URL.Query().Get("outlet_id")

	limitStr := queryParam(r, "limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	events, err := s.sqlite.ListOutletEvents(ctx, outletID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlet events", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}
