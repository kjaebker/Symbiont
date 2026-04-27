package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kjaebker/symbiont/internal/events"
)

// Event represents a Server-Sent Event.
type Event struct {
	Type string
	Data any
}

// Broadcaster manages SSE client connections and broadcasts events.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[string]chan Event
}

// NewBroadcaster creates a new Broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[string]chan Event),
	}
}

// Subscribe registers a new client and returns its event channel.
func (b *Broadcaster) Subscribe(id string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 5)
	b.clients[id] = ch
	return ch
}

// Unsubscribe removes a client and closes its channel.
func (b *Broadcaster) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
}

// Publish sends an event to all connected clients. Slow clients are skipped.
func (b *Broadcaster) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.clients {
		select {
		case ch <- e:
		default:
			// Slow client — skip to avoid blocking.
		}
	}
}

// ClientCount returns the number of connected clients.
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// HandleStream serves an SSE connection. Auth is via ?token= query param.
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	// Validate token from query param (SSE can't use Authorization header with EventSource).
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token query parameter", "unauthorized")
		return
	}
	valid, ta := s.sqlite.ValidateToken(r.Context(), token)
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid token", "unauthorized")
		return
	}
	go func() {
		_ = s.sqlite.TouchToken(context.Background(), ta.ID)
	}()

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "internal_error")
		return
	}

	// Flush headers immediately so the client sees the 200 response.
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	clientID := uuid.New().String()
	ch := s.broadcaster.Subscribe(clientID)
	defer s.broadcaster.Unsubscribe(clientID)

	s.logger.Info("sse client connected", "client_id", clientID)
	defer s.logger.Info("sse client disconnected", "client_id", clientID)

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(w, flusher, evt); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := writeSSE(w, flusher, Event{Type: "heartbeat", Data: nil}); err != nil {
				return
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, evt Event) error {
	fmt.Fprintf(w, "event: %s\n", evt.Type)
	if evt.Data != nil {
		data, err := json.Marshal(evt.Data)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "data: %s\n", data)
	} else {
		fmt.Fprint(w, "data: {}\n")
	}
	fmt.Fprint(w, "\n")
	flusher.Flush()
	return nil
}

// StartSSEPoller runs a background goroutine that polls DuckDB every 10 seconds
// and publishes probe_update and outlet_update events to all SSE clients.
// Events are only broadcast when the data has changed since the last publish.
func (s *Server) StartSSEPoller(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		var lastProbes, lastOutlets []byte
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.broadcaster.ClientCount() == 0 {
					continue
				}
				lastProbes, lastOutlets = s.publishUpdates(ctx, lastProbes, lastOutlets)
			}
		}
	}()
}

func (s *Server) publishUpdates(ctx context.Context, lastProbes, lastOutlets []byte) ([]byte, []byte) {
	probes, err := s.duck.CurrentProbeReadings(ctx)
	if err != nil {
		s.logger.Error("sse: failed to fetch probe readings", "err", err)
	} else if data, err := json.Marshal(probes); err == nil && !bytes.Equal(data, lastProbes) {
		lastProbes = data
		s.broadcaster.Publish(Event{Type: "probe_update", Data: probes})
	}

	outlets, err := s.duck.CurrentOutletStates(ctx)
	if err != nil {
		s.logger.Error("sse: failed to fetch outlet states", "err", err)
	} else if data, err := json.Marshal(outlets); err == nil && !bytes.Equal(data, lastOutlets) {
		lastOutlets = data
		s.broadcaster.Publish(Event{Type: "outlet_update", Data: outlets})
	}

	return lastProbes, lastOutlets
}

// busEventToSSE converts a typed system event to an SSE event for connected
// clients. Returns nil if the event kind should not be forwarded to SSE.
func busEventToSSE(e events.SystemEvent) *Event {
	switch ev := e.(type) {
	case events.EvtFeedModeActivated:
		return &Event{Type: "feed_mode_activated", Data: map[string]any{
			"mode":     ev.Mode,
			"feed_num": ev.FeedNum,
			"at":       ev.OccurredAt(),
		}}
	case events.EvtOutletChanged:
		return &Event{Type: "outlet_changed", Data: map[string]any{
			"outlet_id":    ev.OutletID,
			"name":         ev.Name,
			"prev_state":   ev.PrevState,
			"new_state":    ev.NewState,
			"initiated_by": ev.InitiatedBy,
			"at":           ev.OccurredAt(),
		}}
	case events.EvtAlertFired:
		return &Event{Type: "alert_fired", Data: map[string]any{
			"rule_id":    ev.RuleID,
			"rule_name":  ev.RuleName,
			"probe_name": ev.ProbeName,
			"value":      ev.Value,
			"threshold":  ev.Threshold,
			"severity":   ev.Severity,
			"condition":  ev.Condition,
			"event_id":   ev.EventID,
			"at":         ev.OccurredAt(),
		}}
	case events.EvtAlertCleared:
		return &Event{Type: "alert_cleared", Data: map[string]any{
			"rule_id":    ev.RuleID,
			"rule_name":  ev.RuleName,
			"probe_name": ev.ProbeName,
			"event_id":   ev.EventID,
			"at":         ev.OccurredAt(),
		}}
	case events.EvtObservationRecorded:
		return &Event{Type: "observation_recorded", Data: map[string]any{
			"observation_id": ev.ObservationID,
			"livestock_id":   ev.LivestockID,
			"livestock_name": ev.LivestockName,
			"status":         ev.Status,
			"prev_status":    ev.PrevStatus,
			"has_image":      ev.HasImage,
			"at":             ev.OccurredAt(),
		}}
	case events.EvtLivestockAdded:
		return &Event{Type: "livestock_added", Data: map[string]any{
			"livestock_id": ev.LivestockID,
			"name":         ev.Name,
			"species":      ev.Species,
			"type":         ev.Type,
			"at":           ev.OccurredAt(),
		}}
	case events.EvtLivestockUpdated:
		return &Event{Type: "livestock_updated", Data: map[string]any{
			"livestock_id":   ev.LivestockID,
			"name":           ev.Name,
			"changed_fields": ev.ChangedFields,
			"at":             ev.OccurredAt(),
		}}
	case events.EvtJournalEntryCreated:
		return &Event{Type: "journal_entry_created", Data: map[string]any{
			"entry_id": ev.EntryID,
			"category": ev.Category,
			"source":   ev.Source,
			"at":       ev.OccurredAt(),
		}}
	case events.EvtPollCycleCompleted:
		return &Event{Type: "poll_cycle_completed", Data: map[string]any{
			"duration_ms": ev.DurationMs,
			"probe_count": ev.ProbeCount,
			"at":          ev.OccurredAt(),
		}}
	default:
		return nil
	}
}

// RegisterSSESubscriber wires the Broadcaster as a bus subscriber so that
// typed system events are forwarded to all connected SSE clients.
func (b *Broadcaster) RegisterSSESubscriber(bus *events.Bus) {
	bus.Subscribe("sse", 512, func(ctx context.Context, e events.SystemEvent) {
		if evt := busEventToSSE(e); evt != nil {
			b.Publish(*evt)
		}
	})
}

