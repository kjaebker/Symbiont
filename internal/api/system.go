package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

// HandleSystemLog returns recent structured log lines from the systemd journal.
// If journalctl is unavailable (dev mode), returns an empty list.
func (s *Server) HandleSystemLog(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	args := []string{
		"-u", "symbiont-api",
		"-u", "symbiont-poller",
		"--no-pager", "-o", "json",
		"-n", strconv.Itoa(limit),
	}
	switch r.URL.Query().Get("service") {
	case "api":
		args = []string{"-u", "symbiont-api", "--no-pager", "-o", "json", "-n", strconv.Itoa(limit)}
	case "poller":
		args = []string{"-u", "symbiont-poller", "--no-pager", "-o", "json", "-n", strconv.Itoa(limit)}
	}

	out, err := exec.CommandContext(r.Context(), "journalctl", args...).Output()
	if err != nil {
		// journalctl unavailable or units not found — return empty list gracefully.
		writeJSON(w, http.StatusOK, map[string]any{"lines": []any{}})
		return
	}

	type logLine struct {
		TS      string         `json:"ts"`
		Service string         `json:"service"`
		Level   string         `json:"level"`
		Msg     string         `json:"msg"`
		Fields  map[string]any `json:"fields,omitempty"`
	}

	var lines []logLine
	for _, raw := range bytes.Split(out, []byte("\n")) {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		// Timestamp from __REALTIME_TIMESTAMP (microseconds since epoch, as a string).
		tsStr := ""
		if v, ok := entry["__REALTIME_TIMESTAMP"]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				if micros, err := strconv.ParseInt(s, 10, 64); err == nil {
					tsStr = time.UnixMicro(micros).UTC().Format(time.RFC3339)
				}
			}
		}

		// Service from _SYSTEMD_UNIT.
		service := "api"
		if v, ok := entry["_SYSTEMD_UNIT"]; ok {
			var u string
			if json.Unmarshal(v, &u) == nil && strings.Contains(u, "poller") {
				service = "poller"
			}
		}

		// MESSAGE may be structured slog JSON — parse it if so.
		msgStr := ""
		level := "INFO"
		var fields map[string]any

		if v, ok := entry["MESSAGE"]; ok {
			var msg string
			if json.Unmarshal(v, &msg) == nil {
				var structured map[string]any
				if json.Unmarshal([]byte(msg), &structured) == nil {
					if m, ok := structured["msg"].(string); ok {
						msgStr = m
					}
					if l, ok := structured["level"].(string); ok {
						level = strings.ToUpper(l)
					}
					if t, ok := structured["time"].(string); ok && tsStr == "" {
						tsStr = t
					}
					fields = make(map[string]any)
					for k, v := range structured {
						if k != "msg" && k != "level" && k != "time" && k != "service" {
							fields[k] = v
						}
					}
					if len(fields) == 0 {
						fields = nil
					}
				} else {
					msgStr = msg
				}
			}
		}

		// Fallback level from syslog PRIORITY field.
		if level == "INFO" {
			if v, ok := entry["PRIORITY"]; ok {
				var p string
				if json.Unmarshal(v, &p) == nil {
					switch p {
					case "0", "1", "2", "3":
						level = "ERROR"
					case "4":
						level = "WARN"
					case "7":
						level = "DEBUG"
					}
				}
			}
		}

		if msgStr == "" {
			continue
		}

		lines = append(lines, logLine{
			TS:      tsStr,
			Service: service,
			Level:   level,
			Msg:     msgStr,
			Fields:  fields,
		})
	}

	if lines == nil {
		lines = []logLine{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
