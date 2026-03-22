package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBroadcasterSubscribePublish(t *testing.T) {
	b := NewBroadcaster()

	ch1 := b.Subscribe("c1")
	ch2 := b.Subscribe("c2")

	if b.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", b.ClientCount())
	}

	b.Publish(Event{Type: "test", Data: "hello"})

	select {
	case evt := <-ch1:
		if evt.Type != "test" {
			t.Fatalf("c1: expected type 'test', got %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("c1: timed out waiting for event")
	}

	select {
	case evt := <-ch2:
		if evt.Type != "test" {
			t.Fatalf("c2: expected type 'test', got %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("c2: timed out waiting for event")
	}
}

func TestBroadcasterUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe("c1")
	b.Unsubscribe("c1")

	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", b.ClientCount())
	}

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed")
	}
}

func TestBroadcasterSlowClientSkipped(t *testing.T) {
	b := NewBroadcaster()
	_ = b.Subscribe("slow")

	// Fill the buffer (size 5).
	for i := 0; i < 5; i++ {
		b.Publish(Event{Type: "fill", Data: i})
	}

	// This publish should not block — slow client is skipped.
	done := make(chan struct{})
	go func() {
		b.Publish(Event{Type: "overflow", Data: "should not block"})
		close(done)
	}()

	select {
	case <-done:
		// Good — publish didn't block.
	case <-time.After(time.Second):
		t.Fatal("publish blocked on slow client")
	}

	b.Unsubscribe("slow")
}

func TestSSEStreamEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	// Start a real HTTP server for SSE (httptest.NewRecorder doesn't support streaming).
	ts := httptest.NewServer(env.server.http.Handler)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/stream?token="+env.token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	// Wait for the SSE client to register with the broadcaster.
	for i := 0; i < 50; i++ {
		if env.server.broadcaster.ClientCount() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if env.server.broadcaster.ClientCount() == 0 {
		t.Fatal("SSE client never registered with broadcaster")
	}

	// Publish events periodically so the scanner will see one.
	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			env.server.broadcaster.Publish(Event{Type: "test_event", Data: map[string]string{"msg": "hello"}})
		}
	}()

	// Read SSE lines until we get our test_event.
	type sseResult struct {
		eventLine string
		dataLine  string
		err       error
	}
	resultCh := make(chan sseResult, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		var eventLine, dataLine string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventLine = line
			}
			if strings.HasPrefix(line, "data: ") {
				dataLine = line
				if eventLine == "event: test_event" {
					resultCh <- sseResult{eventLine: eventLine, dataLine: dataLine}
					return
				}
			}
		}
		resultCh <- sseResult{err: scanner.Err()}
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("scanner error: %v", res.err)
		}
		if res.eventLine != "event: test_event" {
			t.Fatalf("expected 'event: test_event', got %q", res.eventLine)
		}
		if !strings.Contains(res.dataLine, "hello") {
			t.Fatalf("expected data to contain 'hello', got %q", res.dataLine)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
	cancel()
}

func TestSSEStreamRejectsInvalidToken(t *testing.T) {
	env := setupTestEnv(t)
	w := env.requestNoAuth(t, "GET", "/api/stream?token=bad-token")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSSEStreamRejectsMissingToken(t *testing.T) {
	env := setupTestEnv(t)
	w := env.requestNoAuth(t, "GET", "/api/stream")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBroadcasterMultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()

	channels := make([]<-chan Event, 5)
	for i := 0; i < 5; i++ {
		channels[i] = b.Subscribe(string(rune('a' + i)))
	}

	b.Publish(Event{Type: "broadcast", Data: "all"})

	for i, ch := range channels {
		select {
		case evt := <-ch:
			if evt.Type != "broadcast" {
				t.Fatalf("subscriber %d: expected type 'broadcast', got %q", i, evt.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}
