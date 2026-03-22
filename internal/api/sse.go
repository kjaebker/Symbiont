package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
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
	valid, id := s.sqlite.ValidateToken(r.Context(), token)
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid token", "unauthorized")
		return
	}
	go func() {
		_ = s.sqlite.TouchToken(context.Background(), id)
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
func (s *Server) StartSSEPoller(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.broadcaster.ClientCount() == 0 {
					continue
				}
				s.publishUpdates(ctx)
			}
		}
	}()
}

func (s *Server) publishUpdates(ctx context.Context) {
	probes, err := s.duck.CurrentProbeReadings(ctx)
	if err != nil {
		s.logger.Error("sse: failed to fetch probe readings", "err", err)
	} else {
		s.broadcaster.Publish(Event{
			Type: "probe_update",
			Data: probes,
		})
	}

	outlets, err := s.duck.CurrentOutletStates(ctx)
	if err != nil {
		s.logger.Error("sse: failed to fetch outlet states", "err", err)
	} else {
		s.broadcaster.Publish(Event{
			Type: "outlet_update",
			Data: outlets,
		})
	}
}

