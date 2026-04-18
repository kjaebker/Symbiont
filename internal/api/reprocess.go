package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// HandleImagesReprocess migrates existing images to deterministic filenames
// (livestock-{id}.jpg, obs-{livestock_id}-{obs_id}.jpg) and regenerates any
// missing thumbnails and original backups.
//
// Query params:
//
//	force=true — regenerate thumbs and originals even if they already exist
func (s *Server) HandleImagesReprocess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	force := r.URL.Query().Get("force") == "true"

	records, err := s.sqlite.ListAllImagesWithIDs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list images", "db_error")
		return
	}

	var migrated, processed, skipped, failed int
	var failures []string

	for _, rec := range records {
		// Compute the canonical deterministic path for this image.
		var targetRel string
		switch rec.Kind {
		case "livestock":
			targetRel = filepath.Join("images", fmt.Sprintf("livestock-%d.jpg", rec.ID))
		case "observation":
			targetRel = filepath.Join("images", fmt.Sprintf("obs-%d-%d.jpg", rec.LivestockID, rec.ID))
		default:
			continue
		}

		absTarget := s.dataFilePath(targetRel)
		absThumb := s.dataFilePath(thumbPath(targetRel))
		absOrig := s.dataFilePath(originalPath(targetRel))

		// Step 1 — rename files if the stored path is not yet deterministic.
		if rec.ImagePath != targetRel {
			absCurrent := s.dataFilePath(rec.ImagePath)

			if err := moveFile(absCurrent, absTarget); err != nil {
				failed++
				failures = append(failures, rec.ImagePath+": rename processed: "+err.Error())
				continue
			}
			// Rename thumb and original if they exist at the old location.
			moveFile(s.dataFilePath(thumbPath(rec.ImagePath)), absThumb)         //nolint:errcheck
			moveFile(s.dataFilePath(originalPath(rec.ImagePath)), absOrig)       //nolint:errcheck

			// Update the DB to reflect the new path.
			var updateErr error
			switch rec.Kind {
			case "livestock":
				updateErr = s.sqlite.SetLivestockImagePath(ctx, rec.ID, &targetRel)
			case "observation":
				updateErr = s.sqlite.UpdateLivestockObservationImagePath(ctx, rec.ID, &targetRel)
			}
			if updateErr != nil {
				failed++
				failures = append(failures, rec.ImagePath+": update db: "+updateErr.Error())
				continue
			}
			migrated++
		}

		// Step 2 — regenerate thumb and/or original if missing (or forced).
		thumbMissing := !fileExists(absThumb)
		origMissing := !fileExists(absOrig)

		if !force && !thumbMissing && !origMissing {
			skipped++
			continue
		}

		data, err := os.ReadFile(absTarget)
		if err != nil {
			failed++
			failures = append(failures, targetRel+": read: "+err.Error())
			continue
		}

		full, thumb, err := processImage(bytesReader(data), ".jpg")
		if err != nil {
			failed++
			failures = append(failures, targetRel+": process: "+err.Error())
			continue
		}

		ok := true
		if force || thumbMissing {
			if err := os.WriteFile(absThumb, thumb, 0o644); err != nil {
				failed++
				failures = append(failures, targetRel+" (thumb): "+err.Error())
				ok = false
			}
		}
		if ok && (force || origMissing) {
			if err := os.WriteFile(absOrig, full, 0o644); err != nil {
				failed++
				failures = append(failures, targetRel+" (original): "+err.Error())
				ok = false
			}
		}
		if ok {
			processed++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":            len(records),
		"migrated":         migrated,
		"processed":        processed,
		"skipped":          skipped,
		"failed":           failed,
		"failures":         failures,
	})
}

// moveFile renames src to dst, falling back to a copy+delete if they are on
// different filesystems.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	os.Remove(src)
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
