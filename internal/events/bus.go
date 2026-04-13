package events

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// SubscriberStats holds runtime statistics for a single bus subscriber.
type SubscriberStats struct {
	Name       string `json:"name"`
	QueueDepth int    `json:"queue_depth"`
	Published  int64  `json:"published"`
	Dropped    int64  `json:"dropped"`
	LastError  string `json:"last_error,omitempty"`
}

// sub is the internal representation of a registered bus subscriber.
type sub struct {
	name      string
	handler   func(context.Context, SystemEvent)
	ch        chan SystemEvent
	done      chan struct{} // closed when the dispatch goroutine exits
	published atomic.Int64
	dropped   atomic.Int64
	lastErr   atomic.Value // stores string
}

// Bus is a publish/subscribe event bus with per-subscriber bounded queues and
// per-subscriber dispatch goroutines. Slow subscribers are isolated — they
// cannot block the publisher or starve other subscribers.
//
// Usage:
//
//	bus := events.NewBus(logger)
//	bus.Subscribe("journal", 256, journalHandler)
//	bus.Subscribe("sse", 512, sseHandler)
//	bus.Start(ctx)
//	// ... publish events ...
//	bus.Drain(5 * time.Second) // on shutdown
type Bus struct {
	mu     sync.Mutex
	subs   []*sub
	logger *slog.Logger
	runCtx context.Context // set by Start
	closed atomic.Bool
}

// NewBus creates a new Bus. logger is used to emit drop warnings.
// Pass nil to use slog.Default().
func NewBus(logger *slog.Logger) *Bus {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bus{logger: logger}
}

// Subscribe registers a new subscriber. name is used for logging and stats.
// capacity is the size of the subscriber's event queue. handler is called in
// the subscriber's dedicated goroutine for every event it receives.
//
// Subscribe may be called before or after Start. If called after Start, the
// subscriber's goroutine is launched immediately using the bus's running context.
func (b *Bus) Subscribe(name string, capacity int, handler func(context.Context, SystemEvent)) {
	s := &sub{
		name:    name,
		handler: handler,
		ch:      make(chan SystemEvent, capacity),
		done:    make(chan struct{}),
	}

	b.mu.Lock()
	b.subs = append(b.subs, s)
	ctx := b.runCtx
	b.mu.Unlock()

	// If the bus is already running, start this subscriber's goroutine now.
	if ctx != nil {
		go b.dispatch(ctx, s)
	}
}

// Start launches one dispatch goroutine per registered subscriber. Calling
// Start more than once panics. Goroutines stop when ctx is cancelled and then
// flush their queues before exiting.
func (b *Bus) Start(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.runCtx != nil {
		panic("events.Bus.Start called more than once")
	}
	b.runCtx = ctx

	for _, s := range b.subs {
		go b.dispatch(ctx, s)
	}
}

// Publish enqueues e to every registered subscriber's queue. It is
// non-blocking: if a subscriber's queue is full, the oldest event is dropped
// and a warning is logged before the new event is enqueued. After Drain is
// called, Publish is a no-op.
func (b *Bus) Publish(e SystemEvent) {
	if b.closed.Load() {
		return
	}

	b.mu.Lock()
	subs := make([]*sub, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	for _, s := range subs {
		b.enqueue(s, e)
	}
}

// enqueue delivers e to s's channel using drop-oldest semantics.
func (b *Bus) enqueue(s *sub, e SystemEvent) {
	select {
	case s.ch <- e:
		s.published.Add(1)
		return
	default:
	}

	// Queue is full — drop the oldest item to make room.
	select {
	case dropped := <-s.ch:
		s.dropped.Add(1)
		b.logger.Warn("event bus: subscriber queue full, dropping oldest event",
			"subscriber", s.name,
			"dropped_kind", dropped.Kind(),
			"new_kind", e.Kind(),
			"queue_depth", len(s.ch),
		)
	default:
		// The worker drained a slot between our two selects.
	}

	// Enqueue the new event. If the channel is somehow still full, drop the
	// new event rather than blocking.
	select {
	case s.ch <- e:
		s.published.Add(1)
	default:
		s.dropped.Add(1)
		b.logger.Warn("event bus: subscriber queue still full after drop, discarding new event",
			"subscriber", s.name,
			"dropped_kind", e.Kind(),
		)
	}
}

// dispatch runs in a dedicated goroutine for each subscriber. It drains the
// subscriber's queue until ctx is cancelled, then flushes any remaining events.
func (b *Bus) dispatch(ctx context.Context, s *sub) {
	defer close(s.done)
	for {
		select {
		case e, ok := <-s.ch:
			if !ok {
				return
			}
			b.call(ctx, s, e)
		case <-ctx.Done():
			// Context cancelled — flush remaining events then exit.
			for {
				select {
				case e := <-s.ch:
					b.call(context.Background(), s, e)
				default:
					return
				}
			}
		}
	}
}

// call invokes the subscriber's handler, recovering from panics.
func (b *Bus) call(ctx context.Context, s *sub, e SystemEvent) {
	defer func() {
		if r := recover(); r != nil {
			errStr := "panic in event handler"
			s.lastErr.Store(errStr)
			b.logger.Error("event bus: subscriber handler panicked",
				"subscriber", s.name,
				"kind", e.Kind(),
				"panic", r,
			)
		}
	}()
	s.handler(ctx, e)
}

// Drain marks the bus as closed (no new events will be published) and waits up
// to timeout for all subscriber dispatch goroutines to finish. Any queues still
// non-empty after the timeout are logged as warnings. Call after the context
// passed to Start is cancelled.
func (b *Bus) Drain(timeout time.Duration) {
	b.closed.Store(true)

	b.mu.Lock()
	subs := make([]*sub, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	deadline := time.Now().Add(timeout)
	for _, s := range subs {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			b.logUndrained(s)
			continue
		}
		select {
		case <-s.done:
		case <-time.After(remaining):
			b.logUndrained(s)
		}
	}
}

func (b *Bus) logUndrained(s *sub) {
	if depth := len(s.ch); depth > 0 {
		b.logger.Warn("event bus: subscriber still has events after drain timeout",
			"subscriber", s.name,
			"remaining", depth,
		)
	}
}

// Stats returns a snapshot of per-subscriber statistics.
func (b *Bus) Stats() []SubscriberStats {
	b.mu.Lock()
	subs := make([]*sub, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	out := make([]SubscriberStats, len(subs))
	for i, s := range subs {
		lastErr := ""
		if v := s.lastErr.Load(); v != nil {
			lastErr, _ = v.(string)
		}
		out[i] = SubscriberStats{
			Name:       s.name,
			QueueDepth: len(s.ch),
			Published:  s.published.Load(),
			Dropped:    s.dropped.Load(),
			LastError:  lastErr,
		}
	}
	return out
}
