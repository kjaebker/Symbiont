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
