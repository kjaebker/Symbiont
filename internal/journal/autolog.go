package journal

import (
	"fmt"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
)

// EntryFromEvent maps a system event to a journal entry that should be
// auto-logged. Returns false if the event type has no journal mapping.
// Adding support for a new auto-logged event type only requires adding a
// case here — no changes to the event bus or the handlers that publish events.
func EntryFromEvent(e events.SystemEvent) (db.JournalEntry, bool) {
	switch e.Type {
	case events.FeedModeActivated:
		name, _ := e.Payload["feed_name"].(string)
		num, _ := e.Payload["feed_number"].(int)
		title := "Fed the fish"
		if name != "" {
			title = "Fed the fish (Feed Mode " + name + ")"
		}
		neutral := "neutral"
		sourceRef := fmt.Sprintf("feed_mode:%d", num)
		return db.JournalEntry{
			Category:  "event",
			Sentiment: &neutral,
			Title:     title,
			Source:    "system",
			SourceRef: &sourceRef,
		}, true
	default:
		return db.JournalEntry{}, false
	}
}
