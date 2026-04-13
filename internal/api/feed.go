package api

import (
	"net/http"
	"time"

	"github.com/kjaebker/symbiont/internal/events"
)

func (s *Server) HandleFeedGet(w http.ResponseWriter, r *http.Request) {
	status, err := s.apex.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch feed status", "apex_error")
		return
	}
	s.logger.Debug("apex feed status raw", "name", status.Feed.Name, "active", status.Feed.Active)

	// The Apex status response encodes feed state differently from the command API.
	// The command API uses name=1-4 and active=0/1, but the status response uses
	// name=256*feedIndex and a non-zero bitmask for active. Normalize to 0/1 and 0-4.
	active := 0
	if status.Feed.Active != 0 {
		active = 1
	}
	name := status.Feed.Name
	if name > 4 {
		// Observed encoding: name = 256 * feedIndex (e.g. Feed A = 256, Feed B = 512)
		if name%256 == 0 && name/256 <= 4 {
			name = name / 256
		} else {
			name = 0
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":   name,
		"active": active,
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

	s.logger.Info("setting feed mode", "name", body.Name, "active", body.Active)

	if err := s.apex.SetFeedMode(r.Context(), body.Name, body.Active); err != nil {
		s.logger.Error("feed mode failed", "err", err, "name", body.Name, "active", body.Active)
		writeError(w, http.StatusBadGateway, "failed to set feed mode: "+err.Error(), "apex_error")
		return
	}

	s.logger.Info("feed mode set", "name", body.Name, "active", body.Active)

	if body.Active && body.Name >= 1 && body.Name <= 4 {
		feedNames := map[int]string{1: "A", 2: "B", 3: "C", 4: "D"}
		s.events.Publish(r.Context(), events.SystemEvent{
			Type: events.FeedModeActivated,
			TS:   time.Now(),
			Payload: map[string]any{
				"feed_name":   feedNames[body.Name],
				"feed_number": body.Name,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":   body.Name,
		"active": body.Active,
	})
}
