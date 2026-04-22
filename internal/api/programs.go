package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// TDataPoint is a single row from an Apex tdata schedule.
// T is seconds since midnight. Ch holds the 13 channel values (0–100).
type TDataPoint struct {
	T  int   `json:"t"`
	Ch []int `json:"ch"`
}

// OutputProgram is the API representation of a single output's Apex program.
type OutputProgram struct {
	DID       string       `json:"did"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Icon      string       `json:"icon"`
	Prog      string       `json:"prog"`
	UpdatedAt time.Time    `json:"updated_at"`
	TData     []TDataPoint `json:"tdata,omitempty"`
}

// parseTData extracts the first tdata block from an Apex program string.
// The program may contain duplicate tdata blocks (normal vs feed-mode); only
// the first block (up to the second "Fallback" line) is returned.
func parseTData(prog string) []TDataPoint {
	var points []TDataPoint
	seenTData := false

	for _, raw := range strings.Split(prog, "\n") {
		line := strings.TrimSpace(raw)

		if strings.HasPrefix(line, "tdata ") {
			seenTData = true
			rest := strings.TrimPrefix(line, "tdata ")
			parts := strings.SplitN(rest, ",", 14)
			if len(parts) < 1 {
				continue
			}

			secs, err := parseHMS(parts[0])
			if err != nil {
				continue
			}

			ch := make([]int, 13)
			for i := 1; i < len(parts) && i <= 13; i++ {
				v, _ := strconv.Atoi(strings.TrimSpace(parts[i]))
				ch[i-1] = v
			}

			points = append(points, TDataPoint{T: secs, Ch: ch})
			continue
		}

		// Stop at the second Fallback line (feed-mode duplicate block).
		if seenTData && strings.HasPrefix(line, "Fallback") {
			break
		}
	}

	return points
}

func parseHMS(s string) (int, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}
	return h*3600 + m*60 + sec, nil
}

// recordToOutputProgram converts a DB record to the API type, parsing tdata.
func recordToOutputProgram(r db.ProgramRecord) OutputProgram {
	p := OutputProgram{
		DID:       r.DID,
		Name:      r.Name,
		Type:      r.Type,
		Icon:      r.Icon,
		Prog:      r.Prog,
		UpdatedAt: r.UpdatedAt,
	}
	if strings.Contains(r.Prog, "\ntdata ") || strings.HasPrefix(r.Prog, "tdata ") {
		p.TData = parseTData(r.Prog)
	}
	return p
}

// syncPrograms fetches /rest/config from the Apex, writes to SQLite, and
// returns the number of programs whose content changed.
func (s *Server) syncPrograms(r *http.Request) (total, changed int, err error) {
	cfg, err := s.apex.Config(r.Context())
	if err != nil {
		return 0, 0, fmt.Errorf("fetching apex config: %w", err)
	}

	records := make([]db.ProgramRecord, 0, len(cfg.OConf))
	for _, oc := range cfg.OConf {
		records = append(records, db.ProgramRecord{
			DID:  oc.DID,
			Name: oc.Name,
			Type: oc.Type,
			Icon: oc.Icon,
			Prog: strings.TrimSpace(oc.Prog),
		})
	}

	n, err := s.sqlite.UpsertOutletPrograms(r.Context(), records)
	if err != nil {
		return 0, 0, fmt.Errorf("storing programs: %w", err)
	}
	return len(records), n, nil
}

// SyncProgramsIfEmpty syncs programs from the Apex on startup only when the
// local DB has no program records yet (first run).
func (s *Server) SyncProgramsIfEmpty(r *http.Request) {
	count, err := s.sqlite.OutletProgramCount(r.Context())
	if err != nil {
		s.logger.Error("checking program count", "err", err)
		return
	}
	if count > 0 {
		return
	}
	total, _, err := s.syncPrograms(r)
	if err != nil {
		s.logger.Error("initial program sync", "err", err)
		return
	}
	s.logger.Info("initial program sync complete", "programs", total)
}

// HandleProgramList returns all outlet programs stored in SQLite along with
// the most recent sync timestamp.
func (s *Server) HandleProgramList(w http.ResponseWriter, r *http.Request) {
	records, err := s.sqlite.ListOutletPrograms(r.Context())
	if err != nil {
		s.logger.Error("listing outlet programs", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list programs", "db_error")
		return
	}

	programs := make([]OutputProgram, len(records))
	var syncedAt *time.Time
	for i, rec := range records {
		programs[i] = recordToOutputProgram(rec)
		if syncedAt == nil || rec.FetchedAt.After(*syncedAt) {
			t := rec.FetchedAt
			syncedAt = &t
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"programs":  programs,
		"synced_at": syncedAt,
	})
}

// HandleProgramSync re-fetches all programs from the Apex and updates SQLite.
func (s *Server) HandleProgramSync(w http.ResponseWriter, r *http.Request) {
	total, changed, err := s.syncPrograms(r)
	if err != nil {
		s.logger.Error("syncing programs", "err", err)
		writeError(w, http.StatusBadGateway, "failed to sync programs from Apex", "apex_error")
		return
	}
	s.logger.Info("program sync complete", "total", total, "changed", changed)
	writeJSON(w, http.StatusOK, map[string]any{
		"total":   total,
		"changed": changed,
	})
}

// startDailyProgramSync runs a daily background sync of Apex programs.
// It fires once per day at a fixed interval from first call.
func (s *Server) startDailyProgramSync(r *http.Request) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				total, changed, err := s.syncPrograms(r)
				if err != nil {
					slog.Error("daily program sync failed", "err", err)
				} else {
					slog.Info("daily program sync complete", "total", total, "changed", changed)
				}
			case <-r.Context().Done():
				return
			}
		}
	}()
}
