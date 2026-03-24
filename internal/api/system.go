package api

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kjaebker/symbiont/internal/backup"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/poller"
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

	// Determine poll_ok: prefer heartbeat file, fall back to DuckDB timestamp.
	pollOK := !lastPoll.IsZero() && time.Since(lastPoll) < 2*s.cfg.PollInterval
	if s.cfg.HeartbeatPath != "" {
		heartbeat := poller.ReadHeartbeat(s.cfg.HeartbeatPath)
		if !heartbeat.IsZero() {
			pollOK = time.Since(heartbeat) < 60*time.Second
		}
	}

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

func (s *Server) HandleBackupTrigger(w http.ResponseWriter, r *http.Request) {
	cfg := backup.Config{
		BackupDir: s.cfg.BackupDir,
		Retain:    7,
	}

	result, err := backup.Run(r.Context(), s.duck, s.sqlite, cfg, s.logger)
	if err != nil {
		// Record failed backup in SQLite.
		errMsg := err.Error()
		job := db.BackupJob{Status: "failed", Error: &errMsg}
		_, _ = s.sqlite.InsertBackupJob(r.Context(), job)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("backup failed: %v", err), "backup_error")
		return
	}

	// Record successful backup in SQLite.
	var path *string
	if len(result.Paths) > 0 {
		p := result.Paths[0]
		path = &p
	}
	job := db.BackupJob{
		Status:    "success",
		Path:      path,
		SizeBytes: &result.SizeBytes,
	}
	_, _ = s.sqlite.InsertBackupJob(r.Context(), job)

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleCleanup(w http.ResponseWriter, r *http.Request) {
	result, err := s.duck.DeleteOldRows(r.Context(), s.cfg.RetentionDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("cleanup failed: %v", err), "cleanup_error")
		return
	}

	s.logger.Info("data cleanup complete",
		"probe_readings", result.ProbeReadings,
		"outlet_states", result.OutletStates,
		"power_events", result.PowerEvents,
		"controller_meta", result.ControllerMeta,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted": map[string]int64{
			"probe_readings":  result.ProbeReadings,
			"outlet_states":   result.OutletStates,
			"power_events":    result.PowerEvents,
			"controller_meta": result.ControllerMeta,
		},
		"retention_days": s.cfg.RetentionDays,
	})
}

func (s *Server) HandleBackupList(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.sqlite.ListBackupJobs(r.Context(), 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch backup jobs", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": jobs})
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
