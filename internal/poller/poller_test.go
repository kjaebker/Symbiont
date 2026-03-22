package poller

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/db"
)

// mockApexClient implements apex.Client for testing.
type mockApexClient struct {
	statusFunc    func(ctx context.Context) (*apex.StatusResponse, error)
	setOutletFunc func(ctx context.Context, did string, state apex.OutletState) error
	statusCalls   int
}

func (m *mockApexClient) Status(ctx context.Context) (*apex.StatusResponse, error) {
	m.statusCalls++
	return m.statusFunc(ctx)
}

func (m *mockApexClient) SetOutlet(ctx context.Context, did string, state apex.OutletState) error {
	if m.setOutletFunc != nil {
		return m.setOutletFunc(ctx, did, state)
	}
	return nil
}


func sampleStatus() *apex.StatusResponse {
	intensity := 100
	return &apex.StatusResponse{
		System: apex.SystemInfo{
			Serial:   "AC5:12345",
			Hostname: "apex",
			Software: "5.10_1B10",
			Hardware: "1.0",
			Type:     "AC5",
			Timezone: "America/Chicago",
			Date:     time.Now().Unix(),
		},
		Inputs: []apex.Input{
			{DID: "base_Temp", Name: "Tmp", Value: 78.2, Type: "Temp"},
			{DID: "base_pH", Name: "pH", Value: 8.21, Type: "pH"},
		},
		Outputs: []apex.Output{
			{DID: "base_Var1", ID: 1, Name: "Return", Type: "outlet", GID: "", Status: []string{"AON", "", "OK", ""}, Intensity: &intensity},
		},
		Power: apex.PowerInfo{
			Failed:   0,
			Restored: 0,
		},
	}
}

func TestPollCycleWritesData(t *testing.T) {
	// Set up temp DuckDB.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	duckDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { duckDB.Close() })

	mock := &mockApexClient{
		statusFunc: func(ctx context.Context) (*apex.StatusResponse, error) {
			return sampleStatus(), nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	p := New(mock, duckDB, 50*time.Millisecond, logger)

	// Run poller for a short duration.
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Millisecond)
	defer cancel()

	p.Run(ctx)

	// Should have polled at least twice (immediate + 1-2 ticks).
	if mock.statusCalls < 2 {
		t.Errorf("expected at least 2 status calls, got %d", mock.statusCalls)
	}

	// Verify data was written.
	readings, err := duckDB.CurrentProbeReadings(context.Background())
	if err != nil {
		t.Fatalf("querying probe readings: %v", err)
	}
	if len(readings) != 2 {
		t.Errorf("expected 2 probe readings, got %d", len(readings))
	}

	outlets, err := duckDB.CurrentOutletStates(context.Background())
	if err != nil {
		t.Fatalf("querying outlet states: %v", err)
	}
	if len(outlets) != 1 {
		t.Errorf("expected 1 outlet state, got %d", len(outlets))
	}
}

func TestPollSkipsOnApexError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	duckDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { duckDB.Close() })

	callCount := 0
	mock := &mockApexClient{
		statusFunc: func(ctx context.Context) (*apex.StatusResponse, error) {
			callCount++
			if callCount == 1 {
				return nil, context.DeadlineExceeded
			}
			return sampleStatus(), nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	p := New(mock, duckDB, 50*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 130*time.Millisecond)
	defer cancel()

	p.Run(ctx)

	// Poller should not have crashed — it should have continued after the error.
	if callCount < 2 {
		t.Errorf("expected at least 2 calls (first fails, second succeeds), got %d", callCount)
	}

	// Data from the successful poll should exist.
	readings, err := duckDB.CurrentProbeReadings(context.Background())
	if err != nil {
		t.Fatalf("querying probe readings: %v", err)
	}
	if len(readings) == 0 {
		t.Error("expected probe readings from successful poll after error recovery")
	}
}
