package api

import (
	"archive/zip"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

func (s *Server) HandleProbeExport(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "probe name is required", "invalid_param")
		return
	}

	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "invalid_param")
		return
	}

	displayName := name
	if cfg, err := s.sqlite.GetProbeConfig(r.Context(), name); err == nil && cfg.DisplayName != nil {
		displayName = sanitizeFilename(*cfg.DisplayName)
	}
	filename := fmt.Sprintf("%s-%s.csv", displayName, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := s.duck.ExportProbeCSV(r.Context(), w, name, from, to); err != nil {
		s.logger.Error("export failed", "err", err, "probe", name)
		// Headers already sent, can't write JSON error.
	}
}

func (s *Server) HandleBulkExport(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "invalid_param")
		return
	}

	names, err := s.duck.ListProbeNames(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list probes", "db_error")
		return
	}

	if len(names) == 0 {
		writeError(w, http.StatusNotFound, "no probe data available", "not_found")
		return
	}

	filename := fmt.Sprintf("symbiont-export-%s.zip", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, name := range names {
		fw, err := zw.Create(fmt.Sprintf("%s.csv", name))
		if err != nil {
			s.logger.Error("zip entry create failed", "err", err, "probe", name)
			return
		}
		if err := s.duck.ExportProbeCSV(r.Context(), fw, name, from, to); err != nil {
			s.logger.Error("export probe failed", "err", err, "probe", name)
			return
		}
	}
}

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// sanitizeFilename replaces non-alphanumeric characters with hyphens and trims.
func sanitizeFilename(s string) string {
	s = nonAlphaNum.ReplaceAllString(s, "-")
	// Trim leading/trailing hyphens.
	for len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "export"
	}
	return s
}

func parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	fromStr := queryParam(r, "from", "")
	toStr := queryParam(r, "to", "")

	var from, to time.Time

	if fromStr == "" {
		from = time.Now().AddDate(0, 0, -7) // Default: last 7 days
	} else {
		var err error
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' timestamp: %s", fromStr)
		}
	}

	if toStr == "" {
		to = time.Now()
	} else {
		var err error
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' timestamp: %s", toStr)
		}
	}

	return from, to, nil
}
