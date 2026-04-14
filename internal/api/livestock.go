package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)

var validLivestockTypes = map[string]bool{
	"fish":        true,
	"coral":       true,
	"invertebrate": true,
	"other":       true,
}

var validLivestockStatuses = map[string]bool{
	"healthy":    true,
	"sick":       true,
	"quarantine": true,
	"deceased":   true,
}

func (s *Server) HandleLivestockList(w http.ResponseWriter, r *http.Request) {
	f := db.LivestockFilter{
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
	}
	items, err := s.sqlite.ListLivestock(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"livestock": items})
}

func (s *Server) HandleLivestockSpecies(w http.ResponseWriter, r *http.Request) {
	species, err := s.sqlite.ListLivestockSpecies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch species", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"species": species})
}

func (s *Server) HandleLivestockGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}
	item, err := s.sqlite.GetLivestockItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) HandleLivestockCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		Name      string  `json:"name"`
		Species   *string `json:"species"`
		Type      string  `json:"type"`
		Quantity  *int    `json:"quantity"`
		Status    *string `json:"status"`
		DateAdded *string `json:"date_added"`
		Notes     *string `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "missing_field")
		return
	}
	if !validLivestockTypes[body.Type] {
		writeError(w, http.StatusBadRequest, "invalid type", "invalid_field")
		return
	}

	quantity := 1
	if body.Quantity != nil {
		quantity = *body.Quantity
	}
	status := "healthy"
	if body.Status != nil {
		if !validLivestockStatuses[*body.Status] {
			writeError(w, http.StatusBadRequest, "invalid status", "invalid_field")
			return
		}
		status = *body.Status
	}

	item := db.LivestockItem{
		Name:      body.Name,
		Species:   body.Species,
		Type:      body.Type,
		Quantity:  quantity,
		Status:    status,
		DateAdded: body.DateAdded,
		Notes:     body.Notes,
	}

	id, err := s.sqlite.InsertLivestockItem(ctx, item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create livestock item", "db_error")
		return
	}

	created, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created item", "db_error")
		return
	}

	// Publish post-commit.
	species := ""
	if item.Species != nil {
		species = *item.Species
	}
	s.events.Publish(events.NewLivestockAdded(id, item.Name, species, item.Type))

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) HandleLivestockUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	existing, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}

	var body struct {
		Name      string  `json:"name"`
		Species   *string `json:"species"`
		Type      string  `json:"type"`
		Quantity  int     `json:"quantity"`
		Status    string  `json:"status"`
		DateAdded *string `json:"date_added"`
		Notes     *string `json:"notes"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "missing_field")
		return
	}
	if !validLivestockTypes[body.Type] {
		writeError(w, http.StatusBadRequest, "invalid type", "invalid_field")
		return
	}
	if !validLivestockStatuses[body.Status] {
		writeError(w, http.StatusBadRequest, "invalid status", "invalid_field")
		return
	}

	updated := db.LivestockItem{
		Name:      body.Name,
		Species:   body.Species,
		Type:      body.Type,
		Quantity:  body.Quantity,
		Status:    body.Status,
		DateAdded: body.DateAdded,
		Notes:     body.Notes,
		ImagePath: existing.ImagePath, // preserve image
	}

	// Detect changed fields for the event payload.
	var changedFields []string
	if body.Name != existing.Name {
		changedFields = append(changedFields, "name")
	}
	if body.Status != existing.Status {
		changedFields = append(changedFields, "status")
	}
	if body.Quantity != existing.Quantity {
		changedFields = append(changedFields, "quantity")
	}

	if err := s.sqlite.UpdateLivestockItem(ctx, id, updated); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update livestock item", "db_error")
		return
	}

	// Publish post-commit.
	s.events.Publish(events.NewLivestockUpdated(id, body.Name, changedFields))

	item, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated item", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) HandleLivestockDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	// Delete image and thumbnail files if present.
	item, err := s.sqlite.GetLivestockItem(r.Context(), id)
	if err == nil && item.ImagePath != nil {
		os.Remove(s.dataFilePath(*item.ImagePath))
		os.Remove(s.dataFilePath(thumbPath(*item.ImagePath)))
	}

	if err := s.sqlite.DeleteLivestockItem(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete livestock item", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) HandleLivestockImageUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	item, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "image too large (max 5MB)", "file_too_large")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing image field", "missing_field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !validExts[ext] {
		writeError(w, http.StatusBadRequest, "image must be JPEG, PNG, or WebP", "invalid_file_type")
		return
	}

	full, thumb, err := processImage(file, ext)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to process image", "invalid_image")
		return
	}

	imagesDir := s.dataFilePath("images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create images directory", "io_error")
		return
	}

	// Always stored as JPEG regardless of upload format.
	filename := fmt.Sprintf("livestock-%d-%d.jpg", id, time.Now().Unix())
	relPath := filepath.Join("images", filename)
	absPath := s.dataFilePath(relPath)
	absThumbPath := s.dataFilePath(thumbPath(relPath))

	if err := os.WriteFile(absPath, full, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image", "io_error")
		return
	}
	if err := os.WriteFile(absThumbPath, thumb, 0o644); err != nil {
		os.Remove(absPath)
		writeError(w, http.StatusInternalServerError, "failed to save thumbnail", "io_error")
		return
	}

	// Remove old image and thumbnail if present.
	if item.ImagePath != nil {
		os.Remove(s.dataFilePath(*item.ImagePath))
		os.Remove(s.dataFilePath(thumbPath(*item.ImagePath)))
	}

	item.ImagePath = &relPath
	if err := s.sqlite.UpdateLivestockItem(ctx, id, *item); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update livestock image", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"image_path":     relPath,
		"thumbnail_path": thumbPath(relPath),
	})
}

func (s *Server) HandleLivestockImageDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	item, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}

	if item.ImagePath != nil {
		os.Remove(s.dataFilePath(*item.ImagePath))
		os.Remove(s.dataFilePath(thumbPath(*item.ImagePath)))
	}

	item.ImagePath = nil
	if err := s.sqlite.UpdateLivestockItem(ctx, id, *item); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update livestock item", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) HandleLivestockObservationList(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	// Verify the livestock item exists.
	if _, err := s.sqlite.GetLivestockItem(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}

	obs, err := s.sqlite.ListLivestockObservations(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch observations", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": obs})
}

func (s *Server) HandleLivestockObservationCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid livestock id", "invalid_param")
		return
	}

	item, err := s.sqlite.GetLivestockItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "livestock item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch livestock item", "db_error")
		return
	}

	var body struct {
		Status *string `json:"status"`
		Note   *string `json:"note"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	statusEmpty := body.Status == nil || *body.Status == ""
	noteEmpty := body.Note == nil || *body.Note == ""
	if statusEmpty && noteEmpty {
		writeError(w, http.StatusBadRequest, "status or note is required", "missing_field")
		return
	}
	if body.Status != nil && *body.Status != "" && !validLivestockStatuses[*body.Status] {
		writeError(w, http.StatusBadRequest, "invalid status", "invalid_field")
		return
	}

	// Normalize empty strings to nil.
	if body.Status != nil && *body.Status == "" {
		body.Status = nil
	}
	if body.Note != nil && *body.Note == "" {
		body.Note = nil
	}

	obsID, err := s.sqlite.InsertLivestockObservation(ctx, db.LivestockObservation{
		LivestockID: id,
		Status:      body.Status,
		Note:        body.Note,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create observation", "db_error")
		return
	}

	// Auto-update item status if the observation specifies a different one.
	prevStatus := item.Status
	if body.Status != nil && *body.Status != item.Status {
		item.Status = *body.Status
		if err := s.sqlite.UpdateLivestockItem(ctx, id, *item); err != nil {
			s.logger.Error("failed to update livestock status from observation", "err", err, "livestock_id", id)
		}
	}

	// Publish post-commit.
	newStatus := ""
	if body.Status != nil {
		newStatus = *body.Status
	}
	s.events.Publish(events.NewObservationRecorded(obsID, id, item.Name, newStatus, prevStatus, false))

	// Return the created observation.
	obs, err := s.sqlite.ListLivestockObservations(ctx, id)
	if err != nil || len(obs) == 0 {
		writeJSON(w, http.StatusCreated, map[string]any{"id": obsID})
		return
	}
	// Find the newly inserted observation by ID.
	for _, o := range obs {
		if o.ID == obsID {
			writeJSON(w, http.StatusCreated, o)
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": obsID})
}

func (s *Server) HandleObservationImageUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	obsID, err := strconv.ParseInt(pathValue(r, "obs_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid observation id", "invalid_param")
		return
	}

	obs, err := s.sqlite.GetLivestockObservation(ctx, obsID)
	if err != nil {
		writeError(w, http.StatusNotFound, "observation not found", "not_found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "image too large (max 5MB)", "file_too_large")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing image field", "missing_field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !validExts[ext] {
		writeError(w, http.StatusBadRequest, "image must be JPEG, PNG, or WebP", "invalid_file_type")
		return
	}

	full, thumb, err := processImage(file, ext)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to process image", "invalid_image")
		return
	}

	imagesDir := s.dataFilePath("images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create images directory", "io_error")
		return
	}

	// Always stored as JPEG regardless of upload format.
	filename := fmt.Sprintf("obs-%d-%d.jpg", obsID, time.Now().Unix())
	relPath := filepath.Join("images", filename)
	absPath := s.dataFilePath(relPath)
	absThumbPath := s.dataFilePath(thumbPath(relPath))

	if err := os.WriteFile(absPath, full, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image", "io_error")
		return
	}
	if err := os.WriteFile(absThumbPath, thumb, 0o644); err != nil {
		os.Remove(absPath)
		writeError(w, http.StatusInternalServerError, "failed to save thumbnail", "io_error")
		return
	}

	// Remove old image and thumbnail if present.
	if obs.ImagePath != nil {
		os.Remove(s.dataFilePath(*obs.ImagePath))
		os.Remove(s.dataFilePath(thumbPath(*obs.ImagePath)))
	}

	if err := s.sqlite.UpdateLivestockObservationImagePath(ctx, obsID, &relPath); err != nil {
		os.Remove(absPath)
		os.Remove(absThumbPath)
		writeError(w, http.StatusInternalServerError, "failed to update observation image", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"image_path":     relPath,
		"thumbnail_path": thumbPath(relPath),
	})
}

func (s *Server) HandleObservationImageDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	obsID, err := strconv.ParseInt(pathValue(r, "obs_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid observation id", "invalid_param")
		return
	}

	obs, err := s.sqlite.GetLivestockObservation(ctx, obsID)
	if err != nil {
		writeError(w, http.StatusNotFound, "observation not found", "not_found")
		return
	}

	if obs.ImagePath != nil {
		os.Remove(s.dataFilePath(*obs.ImagePath))
		os.Remove(s.dataFilePath(thumbPath(*obs.ImagePath)))
	}

	if err := s.sqlite.UpdateLivestockObservationImagePath(ctx, obsID, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update observation", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
