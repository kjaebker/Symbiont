package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/enums"
	"github.com/kjaebker/symbiont/internal/events"
)

// HandleJournalTemplates returns the grouped template catalog.
//
// GET /api/journal/templates
func (s *Server) HandleJournalTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"templates": s.journalTemplates.Grouped(),
	})
}

// HandleJournalList returns a paginated list of journal entries.
// Query params: category, sentiment, from (RFC3339), to (RFC3339), limit (int).
//
// GET /api/journal
func (s *Server) HandleJournalList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := db.JournalFilter{
		Category:  q.Get("category"),
		Sentiment: q.Get("sentiment"),
	}

	if f.Category != "" && !enums.JournalCategories.Has(f.Category) {
		writeError(w, http.StatusBadRequest, "invalid category", "invalid_param")
		return
	}
	if f.Sentiment != "" && !enums.JournalSentiments.Has(f.Sentiment) {
		writeError(w, http.StatusBadRequest, "invalid sentiment", "invalid_param")
		return
	}

	if raw := q.Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' timestamp; use RFC3339", "invalid_param")
			return
		}
		f.From = &t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' timestamp; use RFC3339", "invalid_param")
			return
		}
		f.To = &t
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 500", "invalid_param")
			return
		}
		f.Limit = n
	}

	entries, err := s.sqlite.ListJournalEntries(r.Context(), f)
	if err != nil {
		s.logger.Error("failed to list journal entries", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list journal entries", "db_error")
		return
	}
	if entries == nil {
		entries = []db.JournalEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// HandleJournalCreate creates a new journal entry.
//
// POST /api/journal
func (s *Server) HandleJournalCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category  string  `json:"category"`
		Sentiment *string `json:"sentiment"`
		Title     string  `json:"title"`
		Body      *string `json:"body"`
		Source    string  `json:"source"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required", "invalid_param")
		return
	}
	if !enums.JournalCategories.Has(body.Category) {
		writeError(w, http.StatusBadRequest, "invalid category; must be one of: observation, maintenance, event, milestone", "invalid_param")
		return
	}
	if body.Sentiment != nil && !enums.JournalSentiments.Has(*body.Sentiment) {
		writeError(w, http.StatusBadRequest, "invalid sentiment; must be one of: good, neutral, bad, critical", "invalid_param")
		return
	}

	source := "manual"
	if body.Source == "ai" {
		source = "ai"
	}

	entry := db.JournalEntry{
		Category:  body.Category,
		Sentiment: body.Sentiment,
		Title:     body.Title,
		Body:      body.Body,
		Source:    source,
	}
	id, err := s.sqlite.InsertJournalEntry(r.Context(), entry)
	if err != nil {
		s.logger.Error("failed to insert journal entry", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create journal entry", "db_error")
		return
	}

	created, err := s.sqlite.GetJournalEntry(r.Context(), id)
	if err != nil || created == nil {
		s.logger.Error("failed to fetch created journal entry", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch journal entry", "db_error")
		return
	}

	// Publish post-commit. Only manual entries are published here; system
	// entries written directly via InsertJournalEntry (e.g. by autolog) do
	// not go through this handler, so there is no publish loop.
	s.events.Publish(events.NewJournalEntryCreated(id, body.Category, "manual"))

	writeJSON(w, http.StatusCreated, created)
}

// HandleJournalGet returns a single journal entry by ID.
//
// GET /api/journal/{id}
func (s *Server) HandleJournalGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id", "invalid_param")
		return
	}
	entry, err := s.sqlite.GetJournalEntry(r.Context(), id)
	if err != nil {
		s.logger.Error("failed to get journal entry", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get journal entry", "db_error")
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "journal entry not found", "not_found")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// HandleJournalUpdate updates an existing journal entry.
//
// PUT /api/journal/{id}
func (s *Server) HandleJournalUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id", "invalid_param")
		return
	}

	var body struct {
		Category  string  `json:"category"`
		Sentiment *string `json:"sentiment"`
		Title     string  `json:"title"`
		Body      *string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required", "invalid_param")
		return
	}
	if !enums.JournalCategories.Has(body.Category) {
		writeError(w, http.StatusBadRequest, "invalid category; must be one of: observation, maintenance, event, milestone", "invalid_param")
		return
	}
	if body.Sentiment != nil && !enums.JournalSentiments.Has(*body.Sentiment) {
		writeError(w, http.StatusBadRequest, "invalid sentiment; must be one of: good, neutral, bad, critical", "invalid_param")
		return
	}

	entry := db.JournalEntry{
		Category:  body.Category,
		Sentiment: body.Sentiment,
		Title:     body.Title,
		Body:      body.Body,
	}
	if err := s.sqlite.UpdateJournalEntry(r.Context(), id, entry); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "journal entry not found", "not_found")
			return
		}
		s.logger.Error("failed to update journal entry", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update journal entry", "db_error")
		return
	}

	updated, err := s.sqlite.GetJournalEntry(r.Context(), id)
	if err != nil || updated == nil {
		s.logger.Error("failed to fetch updated journal entry", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch journal entry", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// HandleJournalDelete deletes a journal entry by ID.
//
// DELETE /api/journal/{id}
func (s *Server) HandleJournalDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id", "invalid_param")
		return
	}
	if err := s.sqlite.DeleteJournalEntry(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "journal entry not found", "not_found")
			return
		}
		s.logger.Error("failed to delete journal entry", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete journal entry", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
