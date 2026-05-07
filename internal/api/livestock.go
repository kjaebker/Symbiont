package api

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/enums"
	"github.com/kjaebker/symbiont/internal/events"
)

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
	if !enums.LivestockTypes.Has(body.Type) {
		writeError(w, http.StatusBadRequest, "invalid type", "invalid_field")
		return
	}

	quantity := 1
	if body.Quantity != nil {
		quantity = *body.Quantity
	}
	status := "healthy"
	if body.Status != nil {
		if !enums.LivestockStatuses.Has(*body.Status) {
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
	if !enums.LivestockTypes.Has(body.Type) {
		writeError(w, http.StatusBadRequest, "invalid type", "invalid_field")
		return
	}
	if !enums.LivestockStatuses.Has(body.Status) {
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
	s.handleImageUpload(w, r, s.livestockImageOwner())
}

func (s *Server) HandleLivestockImageDelete(w http.ResponseWriter, r *http.Request) {
	s.handleImageDelete(w, r, s.livestockImageOwner())
}

// HandleLivestockImageEdit accepts a cropped/rotated image blob from the
// frontend editor and replaces the display copy + thumbnail while leaving the
// original backup file untouched.
func (s *Server) HandleLivestockImageEdit(w http.ResponseWriter, r *http.Request) {
	s.handleImageEdit(w, r, s.livestockImageOwner())
}

// HandleLivestockImageReset re-processes the original backup image and
// replaces the display copy + thumbnail with a fresh, unedited version.
func (s *Server) HandleLivestockImageReset(w http.ResponseWriter, r *http.Request) {
	s.handleImageReset(w, r, s.livestockImageOwner())
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
	if body.Status != nil && *body.Status != "" && !enums.LivestockStatuses.Has(*body.Status) {
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
	s.handleImageUpload(w, r, s.observationImageOwner())
}

func (s *Server) HandleObservationImageDelete(w http.ResponseWriter, r *http.Request) {
	s.handleImageDelete(w, r, s.observationImageOwner())
}

// HandleObservationImageEdit accepts a cropped/rotated image blob and replaces
// the display copy + thumbnail while leaving the original backup untouched.
func (s *Server) HandleObservationImageEdit(w http.ResponseWriter, r *http.Request) {
	s.handleImageEdit(w, r, s.observationImageOwner())
}

// HandleObservationImageReset re-processes the original backup image and
// replaces the display copy + thumbnail with a fresh, unedited version.
func (s *Server) HandleObservationImageReset(w http.ResponseWriter, r *http.Request) {
	s.handleImageReset(w, r, s.observationImageOwner())
}
