package api

import (
	"context"
	"net/http"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

type promptType string

const (
	promptTypeOpen   promptType = "open"
	promptTypeThumbs promptType = "thumbs"
	promptTypeRating promptType = "rating"
)

type dailyPrompt struct {
	Type     promptType `json:"type"`
	Question string     `json:"question"`
}

// promptPool is the base pool of daily questions, rotated by day-of-year.
var promptPool = []dailyPrompt{
	{promptTypeOpen, "How does the tank look today?"},
	{promptTypeThumbs, "Everything looking healthy today?"},
	{promptTypeRating, "How would you rate overall tank health today? (1–5)"},
	{promptTypeOpen, "Anything unusual you noticed?"},
	{promptTypeThumbs, "Did your equipment run as expected overnight?"},
	{promptTypeOpen, "How are your corals coloring up lately?"},
	{promptTypeRating, "How are fish behaving today? (1–5)"},
	{promptTypeOpen, "Any changes to water clarity since your last check?"},
	{promptTypeThumbs, "Skimmer performing well?"},
	{promptTypeOpen, "How is feeding response looking?"},
	{promptTypeRating, "How would you rate water parameter stability this week? (1–5)"},
	{promptTypeOpen, "Anything you want to note about the tank today?"},
}

// HandleDailyPromptGet returns today's prompt if it's after noon and not yet answered.
// Returns {"prompt": null} if before noon or already answered.
func (s *Server) HandleDailyPromptGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	// Don't show before noon — tank needs time to settle for the day.
	if now.Hour() < 12 {
		writeJSON(w, http.StatusOK, map[string]any{"prompt": nil})
		return
	}

	// Check if already answered today.
	ref := dailyPromptRef(now)
	answered, err := s.sqlite.HasJournalEntryBySourceRef(ctx, ref)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check prompt status", "db_error")
		return
	}
	if answered {
		writeJSON(w, http.StatusOK, map[string]any{"prompt": nil})
		return
	}

	prompt := s.selectDailyPrompt(ctx, now)
	writeJSON(w, http.StatusOK, map[string]any{"prompt": prompt})
}

// HandleDailyPromptRespond records the user's response as a journal entry.
func (s *Server) HandleDailyPromptRespond(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Question string `json:"question"`
		Response string `json:"response"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}
	if req.Question == "" || req.Response == "" {
		writeError(w, http.StatusBadRequest, "question and response are required", "validation_error")
		return
	}

	now := time.Now()
	ref := dailyPromptRef(now)
	body := req.Response

	entry := db.JournalEntry{
		TS:        now,
		Category:  "observation",
		Title:     req.Question,
		Body:      &body,
		Source:    "system",
		SourceRef: &ref,
	}

	if _, err := s.sqlite.InsertJournalEntry(ctx, entry); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save response", "db_error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

// dailyPromptRef returns the source_ref string used to identify today's prompt entry.
func dailyPromptRef(t time.Time) string {
	return "daily_prompt:" + t.Format("2006-01-02")
}

// selectDailyPrompt chooses a contextual prompt based on recent events, falling
// back to day-of-year rotation through promptPool.
func (s *Server) selectDailyPrompt(ctx context.Context, now time.Time) dailyPrompt {
	since := now.Add(-24 * time.Hour)
	events, _ := s.sqlite.ListAuditEvents(ctx, db.AuditFilter{
		Since: &since,
		Limit: 10,
	})

	for _, e := range events {
		switch e.Kind {
		case "livestock.status_changed":
			return dailyPrompt{promptTypeOpen, "You had some livestock activity recently — how are things looking today?"}
		case "alert.fired":
			return dailyPrompt{promptTypeThumbs, "An alert fired recently — has the issue been resolved?"}
		case "feed_mode.activated":
			return dailyPrompt{promptTypeOpen, "How did feeding go today?"}
		}
	}

	// Fall back to rotating pool indexed by day-of-year.
	return promptPool[now.YearDay()%len(promptPool)]
}
