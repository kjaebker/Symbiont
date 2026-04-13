package events_test

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kjaebker/symbiont/internal/events"
)

// nullLogger returns a logger that discards all output.
func nullLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestBus_DeliverToSubscribers(t *testing.T) {
	bus := events.NewBus(nullLogger())

	var wg sync.WaitGroup
	wg.Add(2)
	var count1, count2 atomic.Int64

	bus.Subscribe("sub1", 16, func(ctx context.Context, e events.SystemEvent) {
		count1.Add(1)
		wg.Done()
	})
	bus.Subscribe("sub2", 16, func(ctx context.Context, e events.SystemEvent) {
		count2.Add(1)
		wg.Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}

	if count1.Load() != 1 {
		t.Errorf("sub1 got %d events, want 1", count1.Load())
	}
	if count2.Load() != 1 {
		t.Errorf("sub2 got %d events, want 1", count2.Load())
	}
}

func TestBus_DropOldest_WhenFull(t *testing.T) {
	bus := events.NewBus(nullLogger())

	// Subscriber with capacity 2 and a blocking handler.
	// Sequence:
	//   1. Publish "A" — goroutine picks it up immediately and blocks on release.
	//   2. Publish "B" — enqueued (queue depth 1).
	//   3. Publish "C" — enqueued (queue depth 2, channel full).
	//   4. Publish "D" — drop-oldest: "B" dropped, "D" enqueued.
	//   → Events that reach the handler: A, C, D (3 total). "B" is dropped.

	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(3) // A + C + D

	bus.Subscribe("slow", 2, func(ctx context.Context, e events.SystemEvent) {
		<-release
		wg.Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1)) // picked up by goroutine, blocks
	time.Sleep(20 * time.Millisecond)               // let the goroutine dequeue "A"

	bus.Publish(events.NewFeedModeActivated("B", 2)) // queue depth 1
	bus.Publish(events.NewFeedModeActivated("C", 3)) // queue depth 2 (full)
	bus.Publish(events.NewFeedModeActivated("D", 4)) // triggers drop-oldest → drops "B"

	close(release) // unblock all handlers

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event processing")
	}

	stats := bus.Stats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 subscriber stat, got %d", len(stats))
	}
	if stats[0].Dropped == 0 {
		t.Error("expected at least one drop, got 0")
	}
}

func TestBus_SlowSubscriberDoesNotStarveOthers(t *testing.T) {
	bus := events.NewBus(nullLogger())

	fastDone := make(chan struct{})
	slowRelease := make(chan struct{})

	bus.Subscribe("fast", 16, func(ctx context.Context, e events.SystemEvent) {
		close(fastDone)
	})
	bus.Subscribe("slow", 1, func(ctx context.Context, e events.SystemEvent) {
		<-slowRelease
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1))

	select {
	case <-fastDone:
		// fast subscriber got the event; slow is still blocked
	case <-time.After(2 * time.Second):
		t.Fatal("fast subscriber was starved by slow subscriber")
	}
	close(slowRelease)
}

func TestBus_DrainFlushesQueue(t *testing.T) {
	bus := events.NewBus(nullLogger())

	var count atomic.Int64
	bus.Subscribe("sub", 64, func(ctx context.Context, e events.SystemEvent) {
		count.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	bus.Start(ctx)

	const n = 10
	for i := 0; i < n; i++ {
		bus.Publish(events.NewFeedModeActivated("A", 1))
	}

	// Cancel context then drain.
	cancel()
	bus.Drain(2 * time.Second)

	if got := count.Load(); got != n {
		t.Errorf("after drain: got %d events processed, want %d", got, n)
	}
}

func TestBus_PublishAfterDrainIsNoop(t *testing.T) {
	bus := events.NewBus(nullLogger())

	var count atomic.Int64
	bus.Subscribe("sub", 16, func(ctx context.Context, e events.SystemEvent) {
		count.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1))
	time.Sleep(20 * time.Millisecond)

	cancel()
	bus.Drain(1 * time.Second)

	before := count.Load()
	bus.Publish(events.NewFeedModeActivated("B", 2)) // should be no-op
	time.Sleep(20 * time.Millisecond)

	if count.Load() != before {
		t.Error("Publish after Drain should be a no-op, but event was delivered")
	}
}

func TestBus_Stats(t *testing.T) {
	bus := events.NewBus(nullLogger())

	bus.Subscribe("sub", 16, func(ctx context.Context, e events.SystemEvent) {
		time.Sleep(5 * time.Millisecond)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1))
	time.Sleep(50 * time.Millisecond) // let it process

	stats := bus.Stats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}
	if stats[0].Name != "sub" {
		t.Errorf("stat name: got %q, want %q", stats[0].Name, "sub")
	}
	if stats[0].Published < 1 {
		t.Errorf("published count: got %d, want >= 1", stats[0].Published)
	}
}

func TestBus_LateSubscribe(t *testing.T) {
	bus := events.NewBus(nullLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx) // start with no subscribers

	var count atomic.Int64
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe after Start.
	bus.Subscribe("late", 16, func(ctx context.Context, e events.SystemEvent) {
		count.Add(1)
		wg.Done()
	})

	bus.Publish(events.NewFeedModeActivated("A", 1))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("late subscriber did not receive event")
	}

	if count.Load() != 1 {
		t.Errorf("got %d events, want 1", count.Load())
	}
}

func TestBus_PanicInHandlerRecovered(t *testing.T) {
	bus := events.NewBus(nullLogger())

	var after atomic.Bool
	var wg sync.WaitGroup
	wg.Add(2)

	bus.Subscribe("panicky", 16, func(ctx context.Context, e events.SystemEvent) {
		defer wg.Done()
		panic("intentional test panic")
	})
	bus.Subscribe("healthy", 16, func(ctx context.Context, e events.SystemEvent) {
		after.Store(true)
		wg.Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	bus.Publish(events.NewFeedModeActivated("A", 1))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out — panic may not have been recovered")
	}

	if !after.Load() {
		t.Error("healthy subscriber did not run after panicky subscriber panicked")
	}

	stats := bus.Stats()
	var panicky events.SubscriberStats
	for _, s := range stats {
		if s.Name == "panicky" {
			panicky = s
		}
	}
	if panicky.LastError == "" {
		t.Error("expected LastError to be set after panic, got empty string")
	}
}
