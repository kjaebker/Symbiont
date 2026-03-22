package api

import (
	"net/http"
	"os"
	"time"
)

func (s *Server) HandleSystemStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta, err := s.duck.ControllerMeta(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch controller meta", "db_error")
		return
	}

	lastPoll, err := s.duck.LastPollTime(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch last poll time", "db_error")
		return
	}

	// Determine poll_ok: last poll within 2x poll interval.
	pollOK := !lastPoll.IsZero() && time.Since(lastPoll) < 2*s.cfg.PollInterval

	// Get DB file sizes.
	duckDBSize := fileSize(s.cfg.DBPath)
	sqliteSize := fileSize(s.cfg.SQLitePath)

	controller := map[string]string{
		"serial":   "",
		"firmware": "",
		"hardware": "",
	}
	if meta != nil {
		controller["serial"] = meta.Serial
		controller["firmware"] = meta.Software
		controller["hardware"] = meta.Hardware
	}

	var lastPollTS string
	if !lastPoll.IsZero() {
		lastPollTS = lastPoll.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"controller": controller,
		"poller": map[string]any{
			"last_poll_ts":          lastPollTS,
			"poll_ok":               pollOK,
			"poll_interval_seconds": int(s.cfg.PollInterval.Seconds()),
		},
		"db": map[string]any{
			"duckdb_size_bytes": duckDBSize,
			"sqlite_size_bytes": sqliteSize,
		},
	})
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
