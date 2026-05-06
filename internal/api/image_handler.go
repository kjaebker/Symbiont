package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kjaebker/symbiont/internal/enums"
)

// imageOwner describes an entity that has a single image attached, with a
// deterministic filename. The owner type abstracts livestock items and
// observations behind one image-handler implementation.
//
// fetchImagePath returns the entity's current image_path (nil if unset). It
// must return sql.ErrNoRows if the id is not found.
//
// canonicalRelPath returns the deterministic relative path for this owner's
// image, e.g. "images/livestock-42.jpg". Called with the *current* entity
// state so observation owners can derive obs-<livestockID>-<obsID>.jpg.
//
// setImagePath persists a new image_path (or nil) for this owner.
type imageOwner struct {
	// label is used in error messages, e.g. "livestock item" / "observation".
	label string

	// pathParam is the URL path-value key holding the entity id.
	pathParam string

	// fetchImagePath returns the entity's current image_path and any extra
	// data needed to build the canonical filename (carried in `extra`). For
	// livestock items, extra is unused. For observations, extra carries the
	// livestock id needed by canonicalRelPath.
	fetchImagePath func(ctx context.Context, id int64) (current *string, extra int64, err error)

	// canonicalRelPath derives the deterministic relative path for the
	// owner. extra is whatever fetchImagePath returned.
	canonicalRelPath func(id, extra int64) string

	// setImagePath persists a new image_path (nil clears it).
	setImagePath func(ctx context.Context, id int64, path *string) error
}

// resolveID parses the entity id from the request path.
func (o *imageOwner) resolveID(r *http.Request) (int64, error) {
	return strconv.ParseInt(pathValue(r, o.pathParam), 10, 64)
}

// fetchOwner reads current state and returns 404/500 directly when needed.
// Returns (currentPath, canonicalPath, ok). ok=false means a response was
// already written.
func (o *imageOwner) fetchOwner(w http.ResponseWriter, r *http.Request) (id int64, currentPath *string, canonicalPath string, ok bool) {
	id, err := o.resolveID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+o.label+" id", "invalid_param")
		return
	}
	currentPath, extra, err := o.fetchImagePath(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, o.label+" not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch "+o.label, "db_error")
		return
	}
	return id, currentPath, o.canonicalRelPath(id, extra), true
}

// readImageFile parses the multipart form and validates the uploaded file.
// On success returns (file, ext). ext is empty when validateExt is false (used
// by Edit, where the canvas always exports JPEG). ok=false means a response
// was already written.
func readImageFile(w http.ResponseWriter, r *http.Request, validateExt bool) (multipartFile, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "image too large (max 30MB)", "file_too_large")
		return multipartFile{}, false
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing image field", "missing_field")
		return multipartFile{}, false
	}
	ext := ".jpg"
	if validateExt {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if !enums.ImageExtensions.Has(ext) {
			file.Close()
			writeError(w, http.StatusBadRequest, "image must be JPEG, PNG, or WebP", "invalid_file_type")
			return multipartFile{}, false
		}
	}
	return multipartFile{file: file, ext: ext}, true
}

type multipartFile struct {
	file interface {
		Read(p []byte) (int, error)
		Close() error
	}
	ext string
}

// writeImageFiles ensures the images dir exists and writes the three byte
// blobs to disk. Pass nil for orig to skip the original (used by Edit/Reset).
func (s *Server) writeImageFiles(w http.ResponseWriter, relPath string, orig, full, thumb []byte) bool {
	imagesDir := s.dataFilePath("images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create images directory", "io_error")
		return false
	}
	if orig != nil {
		if err := os.WriteFile(s.dataFilePath(originalPath(relPath)), orig, 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save original image", "io_error")
			return false
		}
	}
	if err := os.WriteFile(s.dataFilePath(relPath), full, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image", "io_error")
		return false
	}
	if err := os.WriteFile(s.dataFilePath(thumbPath(relPath)), thumb, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save thumbnail", "io_error")
		return false
	}
	return true
}

// removeStale removes display+thumb (and optionally original) files at
// `current` when it differs from `canonical`. Used to clean up legacy
// timestamp-based filenames after an upload moves to the deterministic name.
func (s *Server) removeStale(current *string, canonical string, alsoOriginal bool) {
	if current == nil || *current == canonical {
		return
	}
	os.Remove(s.dataFilePath(*current))
	os.Remove(s.dataFilePath(thumbPath(*current)))
	if alsoOriginal {
		os.Remove(s.dataFilePath(originalPath(*current)))
	}
}

// handleImageUpload services POST /<owner>/{id}/image: original + full + thumb
// are written to disk and the entity's image_path is updated.
func (s *Server) handleImageUpload(w http.ResponseWriter, r *http.Request, o imageOwner) {
	id, currentPath, canonicalPath, ok := o.fetchOwner(w, r)
	if !ok {
		return
	}
	mf, ok := readImageFile(w, r, true)
	if !ok {
		return
	}
	defer mf.file.Close()

	orig, full, thumb, err := processUpload(mf.file, mf.ext)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to process image", "invalid_image")
		return
	}
	if !s.writeImageFiles(w, canonicalPath, orig, full, thumb) {
		return
	}
	s.removeStale(currentPath, canonicalPath, true)
	if err := o.setImagePath(r.Context(), id, &canonicalPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update "+o.label+" image", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"image_path":     canonicalPath,
		"thumbnail_path": thumbPath(canonicalPath),
		"original_path":  originalPath(canonicalPath),
	})
}

// handleImageEdit services POST /<owner>/{id}/image/edit: replaces display
// and thumbnail without touching the original backup.
func (s *Server) handleImageEdit(w http.ResponseWriter, r *http.Request, o imageOwner) {
	id, currentPath, canonicalPath, ok := o.fetchOwner(w, r)
	if !ok {
		return
	}
	mf, ok := readImageFile(w, r, false)
	if !ok {
		return
	}
	defer mf.file.Close()

	full, thumb, err := processImage(mf.file, ".jpg")
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to process image", "invalid_image")
		return
	}
	if !s.writeImageFiles(w, canonicalPath, nil, full, thumb) {
		return
	}
	s.removeStale(currentPath, canonicalPath, false)
	if err := o.setImagePath(r.Context(), id, &canonicalPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update "+o.label+" image", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"image_path":     canonicalPath,
		"thumbnail_path": thumbPath(canonicalPath),
	})
}

// handleImageReset services POST /<owner>/{id}/image/reset: re-derives display
// and thumbnail from the original backup.
func (s *Server) handleImageReset(w http.ResponseWriter, r *http.Request, o imageOwner) {
	id, currentPath, canonicalPath, ok := o.fetchOwner(w, r)
	if !ok {
		return
	}
	if currentPath == nil {
		writeError(w, http.StatusBadRequest, "no image to reset", "no_image")
		return
	}
	// Prefer the canonical original; fall back to the legacy path's original
	// for not-yet-migrated images.
	origAbsPath := s.dataFilePath(originalPath(canonicalPath))
	if _, err := os.Stat(origAbsPath); os.IsNotExist(err) && *currentPath != canonicalPath {
		origAbsPath = s.dataFilePath(originalPath(*currentPath))
	}
	origData, err := os.ReadFile(origAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "no original image available — run 'images reprocess' to backfill", "no_original")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read original image", "io_error")
		return
	}
	full, thumb, err := processImage(bytesReader(origData), ".jpg")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reprocess original image", "invalid_image")
		return
	}
	if !s.writeImageFiles(w, canonicalPath, nil, full, thumb) {
		return
	}
	s.removeStale(currentPath, canonicalPath, false)
	if err := o.setImagePath(r.Context(), id, &canonicalPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update "+o.label+" image", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"image_path":     canonicalPath,
		"thumbnail_path": thumbPath(canonicalPath),
	})
}

// handleImageDelete services DELETE /<owner>/{id}/image: removes display +
// thumb + original from disk and clears the entity's image_path.
func (s *Server) handleImageDelete(w http.ResponseWriter, r *http.Request, o imageOwner) {
	id, currentPath, _, ok := o.fetchOwner(w, r)
	if !ok {
		return
	}
	if currentPath != nil {
		os.Remove(s.dataFilePath(*currentPath))
		os.Remove(s.dataFilePath(thumbPath(*currentPath)))
		os.Remove(s.dataFilePath(originalPath(*currentPath)))
	}
	if err := o.setImagePath(r.Context(), id, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update "+o.label, "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// livestockImageOwner returns the imageOwner config for /api/livestock/{id}/image.
func (s *Server) livestockImageOwner() imageOwner {
	return imageOwner{
		label:     "livestock item",
		pathParam: "id",
		fetchImagePath: func(ctx context.Context, id int64) (*string, int64, error) {
			item, err := s.sqlite.GetLivestockItem(ctx, id)
			if err != nil {
				return nil, 0, err
			}
			return item.ImagePath, 0, nil
		},
		canonicalRelPath: func(id, _ int64) string {
			return filepath.Join("images", fmt.Sprintf("livestock-%d.jpg", id))
		},
		setImagePath: func(ctx context.Context, id int64, path *string) error {
			return s.sqlite.SetLivestockImagePath(ctx, id, path)
		},
	}
}

// observationImageOwner returns the imageOwner config for
// /api/livestock/{id}/observations/{obs_id}/image.
func (s *Server) observationImageOwner() imageOwner {
	return imageOwner{
		label:     "observation",
		pathParam: "obs_id",
		fetchImagePath: func(ctx context.Context, id int64) (*string, int64, error) {
			obs, err := s.sqlite.GetLivestockObservation(ctx, id)
			if err != nil {
				return nil, 0, err
			}
			return obs.ImagePath, obs.LivestockID, nil
		},
		canonicalRelPath: func(id, livestockID int64) string {
			return filepath.Join("images", fmt.Sprintf("obs-%d-%d.jpg", livestockID, id))
		},
		setImagePath: func(ctx context.Context, id int64, path *string) error {
			return s.sqlite.UpdateLivestockObservationImagePath(ctx, id, path)
		},
	}
}
