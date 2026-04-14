package api

import (
	"net/http"
	"os"
	"strings"
)

// HandleImagesReprocess regenerates thumbnails for all livestock and
// observation images that are stored on disk. It reads each original file,
// runs it through processImage, and writes the thumbnail alongside it using
// the -thumb.jpg naming convention. The original file is left untouched.
//
// Query params:
//
//	force=true  — regenerate even if the thumbnail already exists (default: false)
func (s *Server) HandleImagesReprocess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	force := r.URL.Query().Get("force") == "true"

	paths, err := s.sqlite.ListAllImagePaths(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list image paths", "db_error")
		return
	}

	var processed, skipped, failed int
	var failures []string

	for _, relPath := range paths {
		absPath := s.dataFilePath(relPath)
		absThumb := s.dataFilePath(thumbPath(relPath))

		// Skip if thumbnail already exists and force is not set.
		if !force {
			if _, err := os.Stat(absThumb); err == nil {
				skipped++
				continue
			}
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			failed++
			failures = append(failures, relPath+": "+err.Error())
			continue
		}

		// Derive extension from the stored path.
		ext := ""
		if i := strings.LastIndex(relPath, "."); i >= 0 {
			ext = strings.ToLower(relPath[i:])
		}
		if ext == "" {
			ext = ".jpg"
		}

		_, thumb, err := processImage(bytesReader(data), ext)
		if err != nil {
			failed++
			failures = append(failures, relPath+": "+err.Error())
			continue
		}

		if err := os.WriteFile(absThumb, thumb, 0o644); err != nil {
			failed++
			failures = append(failures, relPath+": "+err.Error())
			continue
		}

		processed++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":     len(paths),
		"processed": processed,
		"skipped":   skipped,
		"failed":    failed,
		"failures":  failures,
	})
}
