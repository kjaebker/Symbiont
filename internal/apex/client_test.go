package apex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestLoginSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var body LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding login body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Login != "admin" || body.Password != "1234" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "connect.sid",
			Value: "test-session-id",
			Path:  "/",
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"connect.sid": "test-session-id"})
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234").(*client)

	if err := c.login(context.Background()); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if c.cookie != "test-session-id" {
		t.Errorf("cookie = %q, want %q", c.cookie, "test-session-id")
	}
}

func TestLoginFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "wrong").(*client)

	err := c.login(context.Background())
	if err == nil {
		t.Fatal("expected login error, got nil")
	}
}

func TestLoginFallbackToBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Set-Cookie header — return session ID in body only.
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{ConnectSID: "body-session-id"})
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234").(*client)

	if err := c.login(context.Background()); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if c.cookie != "body-session-id" {
		t.Errorf("cookie = %q, want %q", c.cookie, "body-session-id")
	}
}

func TestStatusSuccess(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/status-response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			http.SetCookie(w, &http.Cookie{
				Name:  "connect.sid",
				Value: "test-session",
			})
			w.WriteHeader(http.StatusOK)
		case "/rest/status":
			cookie := r.Header.Get("Cookie")
			if cookie == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(fixture)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234")
	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status.System.Hostname != "apex" {
		t.Errorf("hostname = %q, want %q", status.System.Hostname, "apex")
	}
	if status.System.Serial != "AC6L:4034" {
		t.Errorf("serial = %q, want %q", status.System.Serial, "AC6L:4034")
	}
	if len(status.Inputs) != 28 {
		t.Errorf("len(inputs) = %d, want 28", len(status.Inputs))
	}
	if len(status.Outputs) != 23 {
		t.Errorf("len(outputs) = %d, want 23", len(status.Outputs))
	}
	if status.Power.Failed != 1770564515 {
		t.Errorf("power.failed = %d, want %d", status.Power.Failed, 1770564515)
	}
}

func TestReauthOn401(t *testing.T) {
	var loginCount atomic.Int32
	var statusCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			loginCount.Add(1)
			http.SetCookie(w, &http.Cookie{
				Name:  "connect.sid",
				Value: "new-session",
			})
			w.WriteHeader(http.StatusOK)
		case "/rest/status":
			count := statusCount.Add(1)
			cookie := r.Header.Get("Cookie")
			// First status request returns 401 (simulating session expiry).
			if count == 1 || cookie != "connect.sid=new-session" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(StatusResponse{
				System: SystemInfo{Hostname: "apex-reauth"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234").(*client)
	// Pre-set an expired cookie to trigger the 401 path.
	c.cookie = "expired-session"

	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() after reauth failed: %v", err)
	}
	if status.System.Hostname != "apex-reauth" {
		t.Errorf("hostname = %q, want %q", status.System.Hostname, "apex-reauth")
	}

	// Should have logged in once (for re-auth) and made 2 status requests.
	if got := loginCount.Load(); got != 1 {
		t.Errorf("login count = %d, want 1", got)
	}
	if got := statusCount.Load(); got != 2 {
		t.Errorf("status request count = %d, want 2", got)
	}
}

func TestSetOutletSuccess(t *testing.T) {
	var capturedBody OutletControlRequest
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/login":
			http.SetCookie(w, &http.Cookie{
				Name:  "connect.sid",
				Value: "test-session",
			})
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut:
			capturedPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234")
	err := c.SetOutlet(context.Background(), "2_1", OutletOn)
	if err != nil {
		t.Fatalf("SetOutlet() failed: %v", err)
	}

	if capturedPath != "/rest/status/outputs/2_1" {
		t.Errorf("path = %q, want %q", capturedPath, "/rest/status/outputs/2_1")
	}
	if capturedBody.DID != "2_1" {
		t.Errorf("body.did = %q, want %q", capturedBody.DID, "2_1")
	}
	if len(capturedBody.Status) != 4 || capturedBody.Status[0] != "ON" {
		t.Errorf("body.status = %v, want [ON, , OK, ]", capturedBody.Status)
	}
	if capturedBody.Type != "outlet" {
		t.Errorf("body.type = %q, want %q", capturedBody.Type, "outlet")
	}
}

func TestSetOutletReauthRetry(t *testing.T) {
	var putCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/login":
			http.SetCookie(w, &http.Cookie{
				Name:  "connect.sid",
				Value: "fresh-session",
			})
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut:
			count := putCount.Add(1)
			if count == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234").(*client)
	c.cookie = "expired"

	err := c.SetOutlet(context.Background(), "2_1", OutletOff)
	if err != nil {
		t.Fatalf("SetOutlet() with reauth failed: %v", err)
	}
	if got := putCount.Load(); got != 2 {
		t.Errorf("PUT count = %d, want 2", got)
	}
}

func TestSetOutletServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			http.SetCookie(w, &http.Cookie{
				Name:  "connect.sid",
				Value: "test-session",
			})
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		}
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL, "admin", "1234")
	err := c.SetOutlet(context.Background(), "bad_did", OutletOn)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}
