package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNtfySendHeaders(t *testing.T) {
	var gotTitle, gotPriority, gotTags, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTitle = r.Header.Get("Title")
		gotPriority = r.Header.Get("Priority")
		gotTags = r.Header.Get("Tags")
		b := make([]byte, 1024)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNtfy(srv.URL)
	err := n.Send(context.Background(), Notification{
		Title:    "Test Alert",
		Body:     "pH is low",
		Priority: "high",
		Tags:     []string{"warning", "pH"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotTitle != "Test Alert" {
		t.Errorf("title = %q, want %q", gotTitle, "Test Alert")
	}
	if gotPriority != "high" {
		t.Errorf("priority = %q, want %q", gotPriority, "high")
	}
	if gotTags != "warning,pH" {
		t.Errorf("tags = %q, want %q", gotTags, "warning,pH")
	}
	if gotBody != "pH is low" {
		t.Errorf("body = %q, want %q", gotBody, "pH is low")
	}
}

func TestNtfyRetryOn500(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNtfy(srv.URL)
	err := n.Send(context.Background(), Notification{Title: "test", Body: "test"})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestNtfyNoRetryOn400(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	n := NewNtfy(srv.URL)
	err := n.Send(context.Background(), Notification{Title: "test", Body: "test"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", got)
	}
}

func TestMultiNotifier(t *testing.T) {
	var count1, count2 atomic.Int32

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count1.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count2.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	multi := NewMulti(NewNtfy(srv1.URL), NewNtfy(srv2.URL))
	err := multi.Send(context.Background(), Notification{Title: "test", Body: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count1.Load() != 1 || count2.Load() != 1 {
		t.Errorf("expected both notifiers to receive, got %d and %d", count1.Load(), count2.Load())
	}
}

func TestNoopNotifier(t *testing.T) {
	n := &NoopNotifier{}
	err := n.Send(context.Background(), Notification{Title: "test"})
	if err != nil {
		t.Fatalf("noop should not error: %v", err)
	}
}
