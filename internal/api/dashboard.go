package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kjaebker/symbiont/internal/db"
)

func (s *Server) HandleDashboardGet(w http.ResponseWriter, r *http.Request) {
	items, err := s.sqlite.ListDashboardItems(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch dashboard layout", "db_error")
		return
	}
	if items == nil {
		items = []db.DashboardItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) HandleDashboardReplace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		Items []db.DashboardItem `json:"items"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	// Validate items.
	validTypes := map[string]bool{"probe": true, "outlet": true, "device": true, "separator": true}
	seen := make(map[string]bool)
	for _, item := range body.Items {
		if !validTypes[item.ItemType] {
			writeError(w, http.StatusBadRequest, "invalid item_type: "+item.ItemType, "invalid_field")
			return
		}
		if item.ItemType == "separator" {
			if item.Label == nil || *item.Label == "" {
				writeError(w, http.StatusBadRequest, "separator items must have a label", "invalid_field")
				return
			}
		} else {
			if item.ReferenceID == nil || *item.ReferenceID == "" {
				writeError(w, http.StatusBadRequest, "non-separator items must have a reference_id", "invalid_field")
				return
			}
			key := item.ItemType + ":" + *item.ReferenceID
			if seen[key] {
				writeError(w, http.StatusBadRequest, "duplicate reference: "+key, "duplicate_ref")
				return
			}
			seen[key] = true
		}
	}

	if err := s.sqlite.ReplaceDashboardLayout(ctx, body.Items); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save dashboard layout", "db_error")
		return
	}

	items, err := s.sqlite.ListDashboardItems(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch dashboard layout", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) HandleDashboardAddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var item db.DashboardItem
	if err := readJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	validTypes := map[string]bool{"probe": true, "outlet": true, "device": true, "separator": true}
	if !validTypes[item.ItemType] {
		writeError(w, http.StatusBadRequest, "invalid item_type", "invalid_field")
		return
	}
	if item.ItemType == "separator" {
		if item.Label == nil || *item.Label == "" {
			writeError(w, http.StatusBadRequest, "separator items must have a label", "invalid_field")
			return
		}
	} else {
		if item.ReferenceID == nil || *item.ReferenceID == "" {
			writeError(w, http.StatusBadRequest, "non-separator items must have a reference_id", "invalid_field")
			return
		}
	}

	id, err := s.sqlite.AddDashboardItem(ctx, item)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "item already on dashboard", "duplicate_ref")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to add dashboard item", "db_error")
		return
	}

	item.ID = id
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) HandleDashboardRemoveItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id", "invalid_param")
		return
	}

	if err := s.sqlite.RemoveDashboardItem(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "dashboard item not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove dashboard item", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
