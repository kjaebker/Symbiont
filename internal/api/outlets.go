package api

import (
	"net/http"
	"sort"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)



type outletResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
	Type        string `json:"type"`
	Intensity   int    `json:"intensity"`
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
	cfgMap := make(map[string]*db.OutletConfig, len(configs))
	for i := range configs {
		cfgMap[configs[i].OutletID] = &configs[i]
	}

	resp := make([]outletResponse, 0, len(outlets))
	for _, o := range outlets {
		displayName := splitCamelCase(o.Name)
		if cfg, ok := cfgMap[o.DID]; ok {
			if cfg.DisplayName != nil {
				displayName = *cfg.DisplayName
			}
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

	// Sort alphabetically by name.
	sort.SliceStable(resp, func(i, j int) bool {
		return resp[i].Name < resp[j].Name
	})

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

	// Publish post-commit. Enrich initiated_by with the token label when set
	// so events distinguish "api:claude-mobile" from "api:default".
	prevState := ""
	if fromState != nil {
		prevState = *fromState
	}
	name := derefStr(outletName, did)
	initiatedBy := "api"
	if label := TokenLabelFromContext(ctx); label != "" && label != "default" {
		initiatedBy = "api:" + label
	}
	s.events.Publish(events.NewOutletChanged(did, name, prevState, body.State, initiatedBy))

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    did,
		"name":  derefStr(outletName, did),
		"state": body.State,
	})
}
