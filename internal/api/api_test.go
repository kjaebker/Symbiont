package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"log/slog"
	"strconv"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
)

// mockApexClient implements apex.Client for testing.
type mockApexClient struct {
	setOutletErr error
}

func (m *mockApexClient) Status(ctx context.Context) (*apex.StatusResponse, error) {
	return nil, nil
}

func (m *mockApexClient) SetOutlet(ctx context.Context, did string, state apex.OutletState) error {
	return m.setOutletErr
}

func (m *mockApexClient) SetOutletAuto(ctx context.Context, outletName string) error {
	return m.setOutletErr
}


type testEnv struct {
	server *Server
	token  string
	duck   *db.DuckDB
	sqlite *db.SQLiteDB
	mock   *mockApexClient
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// DuckDB on temp file.
	tmpDir := t.TempDir()
	duckDB, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { duckDB.Close() })

	// SQLite in memory.
	sqliteDB, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { sqliteDB.Close() })

	// Create a test token.
	token, err := sqliteDB.InsertToken(context.Background(), "test")
	if err != nil {
		t.Fatalf("inserting token: %v", err)
	}

	mock := &mockApexClient{}
	logger := slog.Default()
	cfg := &config.Config{
		APIPort:      "0",
		PollInterval: 10 * time.Second,
		DBPath:       filepath.Join(tmpDir, "test.db"),
		SQLitePath:   ":memory:",
	}

	server := New(cfg, duckDB, sqliteDB, mock, logger)

	return &testEnv{
		server: server,
		token:  token,
		duck:   duckDB,
		sqlite: sqliteDB,
		mock:   mock,
	}
}

func (e *testEnv) request(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.server.http.Handler.ServeHTTP(w, req)
	return w
}

func (e *testEnv) requestNoAuth(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	e.server.http.Handler.ServeHTTP(w, req)
	return w
}

func seedTestData(t *testing.T, duck *db.DuckDB) {
	t.Helper()
	ctx := context.Background()
	ts := time.Now()

	intensity := 100
	status := &apex.StatusResponse{
		System: apex.SystemInfo{
			Serial: "AC5:12345", Hostname: "apex", Software: "5.10_1B10",
			Hardware: "1.0", Type: "AC5", Timezone: "America/Chicago", Date: ts.Unix(),
		},
		Inputs: []apex.Input{
			{DID: "base_Temp", Name: "Tmp", Value: 78.2, Type: "Temp"},
			{DID: "base_pH", Name: "pH", Value: 8.21, Type: "pH"},
		},
		Outputs: []apex.Output{
			{DID: "base_Var1", ID: 1, Name: "Return", Type: "outlet", Status: []string{"AON", "", "OK", ""}, Intensity: &intensity},
		},
		Power: apex.PowerInfo{Failed: 0, Restored: 0},
	}

	if err := duck.WritePollCycle(ctx, ts, status); err != nil {
		t.Fatalf("seeding test data: %v", err)
	}
}

// --- Auth Tests ---

func TestAuthRejectsMissingToken(t *testing.T) {
	env := setupTestEnv(t)
	w := env.requestNoAuth(t, "GET", "/api/probes")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	env := setupTestEnv(t)
	req := httptest.NewRequest("GET", "/api/probes", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	env.server.http.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthAcceptsValidToken(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)
	w := env.request(t, "GET", "/api/probes", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Probe Tests ---

func TestProbeList(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "GET", "/api/probes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	probes, ok := resp["probes"].([]any)
	if !ok || len(probes) != 2 {
		t.Fatalf("expected 2 probes, got %v", resp["probes"])
	}
}

func TestProbeStatusComputation(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		cfg    *probeConfigLookup
		expect string
	}{
		{"no config", 78.0, nil, "unknown"},
		{"no thresholds", 78.0, &probeConfigLookup{}, "unknown"},
		{"normal", 78.0, &probeConfigLookup{minNormal: ptr(76.0), maxNormal: ptr(82.0)}, "normal"},
		{"warning low", 75.0, &probeConfigLookup{minNormal: ptr(76.0), maxNormal: ptr(82.0)}, "warning"},
		{"warning high", 83.0, &probeConfigLookup{minNormal: ptr(76.0), maxNormal: ptr(82.0)}, "warning"},
		{"critical low", 70.0, &probeConfigLookup{
			minNormal: ptr(76.0), maxNormal: ptr(82.0),
			minWarning: ptr(74.0), maxWarning: ptr(84.0),
		}, "critical"},
		{"critical high", 85.0, &probeConfigLookup{
			minNormal: ptr(76.0), maxNormal: ptr(82.0),
			minWarning: ptr(74.0), maxWarning: ptr(84.0),
		}, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeProbeStatus(tt.value, tt.cfg)
			if got != tt.expect {
				t.Errorf("computeProbeStatus(%v) = %q, want %q", tt.value, got, tt.expect)
			}
		})
	}
}

func TestProbeHistory(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "GET", "/api/probes/Tmp/history?interval=10s", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["probe"] != "Tmp" {
		t.Errorf("expected probe 'Tmp', got %v", resp["probe"])
	}
	if resp["interval"] != "10s" {
		t.Errorf("expected interval '10s', got %v", resp["interval"])
	}
}

func TestProbeHistoryAutoInterval(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expect   string
	}{
		{1 * time.Hour, "10s"},
		{6 * time.Hour, "1m"},
		{2 * 24 * time.Hour, "5m"},
		{7 * 24 * time.Hour, "15m"},
		{30 * 24 * time.Hour, "1h"},
		{90 * 24 * time.Hour, "1d"},
	}

	for _, tt := range tests {
		got := autoInterval(tt.duration)
		if got != tt.expect {
			t.Errorf("autoInterval(%v) = %q, want %q", tt.duration, got, tt.expect)
		}
	}
}

// --- Outlet Tests ---

func TestOutletList(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "GET", "/api/outlets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	outlets, ok := resp["outlets"].([]any)
	if !ok || len(outlets) != 1 {
		t.Fatalf("expected 1 outlet, got %v", resp["outlets"])
	}
}

func TestOutletSetSuccess(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "PUT", "/api/outlets/base_Var1", map[string]string{"state": "OFF"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify event was logged.
	events, err := env.sqlite.ListOutletEvents(context.Background(), "base_Var1", 10)
	if err != nil {
		t.Fatalf("listing events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].InitiatedBy != "api" {
		t.Errorf("expected initiated_by 'api', got %q", events[0].InitiatedBy)
	}
}

func TestOutletSetApexError(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)
	env.mock.setOutletErr = context.DeadlineExceeded

	w := env.request(t, "PUT", "/api/outlets/base_Var1", map[string]string{"state": "ON"})
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOutletSetInvalidState(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, "PUT", "/api/outlets/base_Var1", map[string]string{"state": "INVALID"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOutletEvents(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	toState := "ON"
	env.sqlite.InsertOutletEvent(ctx, db.OutletEvent{
		OutletID: "base_Var1", ToState: toState, InitiatedBy: "api",
	})

	w := env.request(t, "GET", "/api/outlets/events?outlet_id=base_Var1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	events, ok := resp["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected 1 event, got %v", resp["events"])
	}
}

// --- System Test ---

func TestSystemStatus(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "GET", "/api/system", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	controller, ok := resp["controller"].(map[string]any)
	if !ok {
		t.Fatal("missing controller in response")
	}
	if controller["serial"] != "AC5:12345" {
		t.Errorf("expected serial 'AC5:12345', got %v", controller["serial"])
	}

	poller, ok := resp["poller"].(map[string]any)
	if !ok {
		t.Fatal("missing poller in response")
	}
	if poller["poll_ok"] != true {
		t.Errorf("expected poll_ok true, got %v", poller["poll_ok"])
	}
}

// --- Config Tests ---

func TestProbeConfigCRUD(t *testing.T) {
	env := setupTestEnv(t)

	// Create.
	w := env.request(t, "PUT", "/api/config/probes/Tmp", map[string]any{
		"display_name":  "Temperature",
		"display_order": 1,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List.
	w = env.request(t, "GET", "/api/config/probes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	configs, ok := resp["configs"].([]any)
	if !ok || len(configs) != 1 {
		t.Fatalf("expected 1 config, got %v", resp["configs"])
	}
}

func TestOutletConfigCRUD(t *testing.T) {
	env := setupTestEnv(t)

	w := env.request(t, "PUT", "/api/config/outlets/base_Var1", map[string]any{
		"display_name":  "Return Pump",
		"display_order": 1,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = env.request(t, "GET", "/api/config/outlets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Alert Tests ---

func TestAlertCRUD(t *testing.T) {
	env := setupTestEnv(t)

	// Create.
	w := env.request(t, "POST", "/api/alerts", map[string]any{
		"probe_name":     "Tmp",
		"condition":      "outside_range",
		"threshold_low":  76.0,
		"threshold_high": 82.0,
		"severity":       "warning",
		"enabled":        true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(float64)

	// List.
	w = env.request(t, "GET", "/api/alerts", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update.
	w = env.request(t, "PUT", "/api/alerts/1", map[string]any{
		"probe_name":     "Tmp",
		"condition":      "above",
		"threshold_high": 84.0,
		"severity":       "critical",
		"enabled":        true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete.
	w = env.request(t, "DELETE", "/api/alerts/"+intToStr(int(id)), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlertValidation(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"invalid condition", map[string]any{"probe_name": "Tmp", "condition": "invalid", "severity": "warning"}},
		{"invalid severity", map[string]any{"probe_name": "Tmp", "condition": "above", "threshold_high": 80.0, "severity": "invalid"}},
		{"missing threshold", map[string]any{"probe_name": "Tmp", "condition": "above", "severity": "warning"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.request(t, "POST", "/api/alerts", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

// --- Token Tests ---

func TestTokenLifecycle(t *testing.T) {
	env := setupTestEnv(t)

	// Create.
	w := env.request(t, "POST", "/api/tokens", map[string]string{"label": "new-token"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List.
	w = env.request(t, "GET", "/api/tokens", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	tokens := resp["tokens"].([]any)
	if len(tokens) != 2 { // test token + new token
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	// Delete the new one (id=2).
	w = env.request(t, "DELETE", "/api/tokens/2", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Helpers ---

func ptr(f float64) *float64 { return &f }

func intToStr(i int) string {
	return strconv.Itoa(i)
}
