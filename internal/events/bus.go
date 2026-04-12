// Package events provides a lightweight in-process event bus for system-generated
// events. Handlers can publish typed events without importing the journal or
// notification packages; subscribers wire up the side effects at startup.
package events

import (
	"context"
	"sync"
	"time"
)

// EventType identifies the kind of system event that occurred.
type EventType string

const (
	// FeedModeActivated is published when a Neptune Apex feed mode is turned on.
	// Payload keys: "feed_name" (string, e.g. "A"), "feed_number" (int, 1–4).
	FeedModeActivated EventType = "feed_mode_activated"

	// Future event types can be added here without touching subscriber code.
	// Examples:
	//   AlertFired    EventType = "alert_fired"
	//   AlertCleared  EventType = "alert_cleared"
	//   OutletChanged EventType = "outlet_changed"
)

// SystemEvent carries the type and optional payload for a system event.
type SystemEvent struct {
	Type    EventType
	TS      time.Time
	Payload map[string]any
}

// Bus is a simple publish/subscribe event bus. Listeners are called
// asynchronously in their own goroutines so that publishing never blocks
// the caller. Each listener receives the same context that was passed to
// Publish, so cancellation propagates naturally.
type Bus struct {
	mu        sync.RWMutex
	listeners []func(context.Context, SystemEvent)
}

// NewBus returns a new, empty Bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers fn to be called on every future published event.
// fn is called in a new goroutine; it must not assume sequential execution
// relative to other listeners.
func (b *Bus) Subscribe(fn func(context.Context, SystemEvent)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners = append(b.listeners, fn)
}

// Publish dispatches e to all registered listeners, each in its own goroutine.
// It returns immediately — delivery is best-effort and non-blocking.
func (b *Bus) Publish(ctx context.Context, e SystemEvent) {
	b.mu.RLock()
	listeners := make([]func(context.Context, SystemEvent), len(b.listeners))
	copy(listeners, b.listeners)
	b.mu.RUnlock()

	for _, fn := range listeners {
		go fn(ctx, e)
	}
}
