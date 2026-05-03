package journal

import (
	"fmt"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)



// EntryFromEvent maps a typed system event to a journal entry that should be
// auto-logged. Returns false if this event kind has no journal mapping.
// Adding support for a new auto-logged event type only requires adding a
// case here — no changes to the event bus or the handlers that publish events.
func EntryFromEvent(e events.SystemEvent) (db.JournalEntry, bool) {
	switch ev := e.(type) {
	case events.EvtFeedModeActivated:
		title := "Fed the fish"
		if ev.Mode != "" {
			title = "Fed the fish (Feed Mode " + ev.Mode + ")"
		}
		neutral := "neutral"
		sourceRef := fmt.Sprintf("feed_mode:%d", ev.FeedNum)
		return db.JournalEntry{
			Category:  "event",
			Sentiment: &neutral,
			Title:     title,
			Source:    "system",
			SourceRef: &sourceRef,
		}, true

	case events.EvtObservationRecorded:
		// Only autolog significant observations:
		//   - First observation (PrevStatus == ""), or
		//   - Status transitions to "sick" or "deceased", or
		//   - Image attached (HasImage == true)
		significant := ev.PrevStatus == "" ||
			ev.Status == "sick" || ev.Status == "deceased" ||
			ev.HasImage
		if !significant {
			return db.JournalEntry{}, false
		}

		title := fmt.Sprintf("Observation: %s", ev.LivestockName)
		if ev.Status != "" && ev.Status != ev.PrevStatus {
			title = fmt.Sprintf("%s status changed to %s", ev.LivestockName, ev.Status)
		}
		bad := "bad"
		good := "good"
		neutral := "neutral"
		sentiment := &neutral
		if ev.Status == "sick" || ev.Status == "deceased" {
			sentiment = &bad
		} else if ev.Status == "healthy" && ev.PrevStatus != "" {
			sentiment = &good
		}
		sourceRef := fmt.Sprintf("observation:%d", ev.ObservationID)
		return db.JournalEntry{
			Category:  "observation",
			Sentiment: sentiment,
			Title:     title,
			Source:    "system",
			SourceRef: &sourceRef,
		}, true

	case events.EvtDoseLogged:
		neutral := "neutral"
		title := fmt.Sprintf("Dosed %.4g %s %s %s", ev.Amount, ev.Unit, ev.Brand, ev.Name)
		sourceRef := fmt.Sprintf("dose_log:%d", ev.LogID)
		return db.JournalEntry{
			Category:  "maintenance",
			Sentiment: &neutral,
			Title:     title,
			Source:    "system",
			SourceRef: &sourceRef,
		}, true

	case events.EvtMaintenanceCompleted:
		neutral := "neutral"
		title := fmt.Sprintf("%s completed", ev.TaskName)
		sourceRef := fmt.Sprintf("maintenance_log:%d", ev.LogID)
		entry := db.JournalEntry{
			Category:  "maintenance",
			Sentiment: &neutral,
			Title:     title,
			Source:    "system",
			SourceRef: &sourceRef,
		}
		if ev.Notes != nil && *ev.Notes != "" {
			entry.Body = ev.Notes
		}
		return entry, true

	default:
		return db.JournalEntry{}, false
	}
}
