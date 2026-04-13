package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// HandleAuditEventList returns paginated audit events from the events table.
//
// GET /api/events?kind=&since=&limit=
func (s *Server) HandleAuditEventList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := db.AuditFilter{Kind: q.Get("kind")}

	if raw := q.Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'since' timestamp; use RFC3339", "invalid_param")
			return
		}
		f.Since = &t
	}

	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer", "invalid_param")
			return
		}
		f.Limit = n
	}

	evts, err := s.sqlite.ListAuditEvents(r.Context(), f)
	if err != nil {
		s.logger.Error("failed to list audit events", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list events", "db_error")
		return
	}
	if evts == nil {
		evts = []db.AuditEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": evts})
}

// HandleEventBusStats returns per-subscriber queue depth, publish, and drop counts.
//
// GET /api/events/stats
func (s *Server) HandleEventBusStats(w http.ResponseWriter, r *http.Request) {
	stats := s.events.Stats()
	writeJSON(w, http.StatusOK, map[string]any{"subscribers": stats})
}
