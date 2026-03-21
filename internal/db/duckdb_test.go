package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
)

func testDB(t *testing.T) *DuckDB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestCreateSchemaIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db.Close()

	// Open again — CreateSchema runs again, should not error
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second open (idempotent schema): %v", err)
	}
	db2.Close()
}

func sampleStatus() *apex.StatusResponse {
	intensity := 75
	return &apex.StatusResponse{
		System: apex.SystemInfo{
			Hostname: "apex",
			Software: "5.10_8C18",
			Hardware: "1.0",
			Serial:   "AC5:12345",
			Type:     "A3",
			Timezone: "America/Chicago",
		},
		Inputs: []apex.Input{
			{DID: "base_Temp", Type: "Temp", Name: "Temp", Value: 78.2},
			{DID: "base_pH", Type: "pH", Name: "pH", Value: 8.15},
			{DID: "base_Amps", Type: "Amps", Name: "Amps_1", Value: 0.3},
		},
		Outputs: []apex.Output{
			{
				DID:       "base_outlet_1",
				ID:        1,
				Name:      "Heater",
				Type:      "outlet",
				Status:    []string{"AON", "", "OK", ""},
				Intensity: nil,
			},
			{
				DID:       "base_outlet_2",
				ID:        2,
				Name:      "Light",
				Type:      "outlet",
				Status:    []string{"ON", "75", "OK", ""},
				Intensity: &intensity,
			},
		},
		Power: apex.PowerInfo{
			Failed:   1700000000,
			Restored: 1700000060,
		},
	}
}

func TestWritePollCycle(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	status := sampleStatus()

	if err := db.WritePollCycle(ctx, ts, status); err != nil {
		t.Fatalf("WritePollCycle: %v", err)
	}

	// Verify probe readings were written
	var probeCount int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM probe_readings").Scan(&probeCount); err != nil {
		t.Fatalf("counting probe readings: %v", err)
	}
	if probeCount != 3 {
		t.Errorf("expected 3 probe readings, got %d", probeCount)
	}

	// Verify outlet states were written
	var outletCount int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM outlet_states").Scan(&outletCount); err != nil {
		t.Fatalf("counting outlet states: %v", err)
	}
	if outletCount != 2 {
		t.Errorf("expected 2 outlet states, got %d", outletCount)
	}

	// Verify power events were written
	var powerCount int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM power_events").Scan(&powerCount); err != nil {
		t.Fatalf("counting power events: %v", err)
	}
	if powerCount != 2 {
		t.Errorf("expected 2 power events, got %d", powerCount)
	}

	// Verify controller meta was written
	var metaCount int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM controller_meta").Scan(&metaCount); err != nil {
		t.Fatalf("counting controller meta: %v", err)
	}
	if metaCount != 1 {
		t.Errorf("expected 1 controller meta, got %d", metaCount)
	}
}

func TestWritePowerEventsDeduplication(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	power := apex.PowerInfo{
		Failed:   1700000000,
		Restored: 1700000060,
	}

	ts1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC)

	// Write same power events twice with different poll timestamps
	if err := db.WritePowerEvents(ctx, ts1, power); err != nil {
		t.Fatalf("first WritePowerEvents: %v", err)
	}
	if err := db.WritePowerEvents(ctx, ts2, power); err != nil {
		t.Fatalf("second WritePowerEvents: %v", err)
	}

	var count int
	if err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM power_events").Scan(&count); err != nil {
		t.Fatalf("counting power events: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 deduplicated power events (1 failed + 1 restored), got %d", count)
	}
}

func TestCurrentProbeReadings(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ts1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC)

	// Write two rounds of readings
	inputs1 := []apex.Input{
		{DID: "base_Temp", Type: "Temp", Name: "Temp", Value: 78.0},
		{DID: "base_pH", Type: "pH", Name: "pH", Value: 8.10},
	}
	inputs2 := []apex.Input{
		{DID: "base_Temp", Type: "Temp", Name: "Temp", Value: 78.5},
		{DID: "base_pH", Type: "pH", Name: "pH", Value: 8.20},
	}

	if err := db.WriteProbeReadings(ctx, ts1, inputs1); err != nil {
		t.Fatalf("first WriteProbeReadings: %v", err)
	}
	if err := db.WriteProbeReadings(ctx, ts2, inputs2); err != nil {
		t.Fatalf("second WriteProbeReadings: %v", err)
	}

	readings, err := db.CurrentProbeReadings(ctx)
	if err != nil {
		t.Fatalf("CurrentProbeReadings: %v", err)
	}

	if len(readings) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(readings))
	}

	// Results are ordered by probe_name, so pH comes first, then Temp
	for _, r := range readings {
		switch r.Name {
		case "Temp":
			if r.Value != 78.5 {
				t.Errorf("expected Temp=78.5, got %f", r.Value)
			}
		case "pH":
			if r.Value != 8.20 {
				t.Errorf("expected pH=8.20, got %f", r.Value)
			}
		default:
			t.Errorf("unexpected probe name: %s", r.Name)
		}
	}
}

func TestProbeHistory(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Insert readings across a 30-minute window
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		ts := base.Add(time.Duration(i) * 5 * time.Minute)
		inputs := []apex.Input{
			{DID: "base_Temp", Type: "Temp", Name: "Temp", Value: 78.0 + float64(i)*0.1},
		}
		if err := db.WriteProbeReadings(ctx, ts, inputs); err != nil {
			t.Fatalf("WriteProbeReadings at %v: %v", ts, err)
		}
	}

	from := base
	to := base.Add(30 * time.Minute)
	points, err := db.ProbeHistory(ctx, "Temp", from, to, "10 minutes")
	if err != nil {
		t.Fatalf("ProbeHistory: %v", err)
	}

	// 6 readings across 30 minutes bucketed into 10-minute intervals = 3 buckets
	if len(points) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(points))
	}
}

func TestCurrentOutletStates(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ts1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC)

	intensity := 50
	outputs1 := []apex.Output{
		{DID: "base_outlet_1", ID: 1, Name: "Heater", Type: "outlet", Status: []string{"ON", "", "OK", ""}},
	}
	outputs2 := []apex.Output{
		{DID: "base_outlet_1", ID: 1, Name: "Heater", Type: "outlet", Status: []string{"AOF", "", "OK", ""}, Intensity: &intensity},
	}

	if err := db.WriteOutletStates(ctx, ts1, outputs1); err != nil {
		t.Fatalf("first WriteOutletStates: %v", err)
	}
	if err := db.WriteOutletStates(ctx, ts2, outputs2); err != nil {
		t.Fatalf("second WriteOutletStates: %v", err)
	}

	states, err := db.CurrentOutletStates(ctx)
	if err != nil {
		t.Fatalf("CurrentOutletStates: %v", err)
	}

	if len(states) != 1 {
		t.Fatalf("expected 1 outlet state, got %d", len(states))
	}

	if states[0].State != "AOF" {
		t.Errorf("expected state AOF, got %s", states[0].State)
	}
	if states[0].Intensity != 50 {
		t.Errorf("expected intensity 50, got %d", states[0].Intensity)
	}
}

func TestLastPollTime(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ts1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC)

	inputs := []apex.Input{
		{DID: "base_Temp", Type: "Temp", Name: "Temp", Value: 78.0},
	}

	if err := db.WriteProbeReadings(ctx, ts1, inputs); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := db.WriteProbeReadings(ctx, ts2, inputs); err != nil {
		t.Fatalf("second write: %v", err)
	}

	last, err := db.LastPollTime(ctx)
	if err != nil {
		t.Fatalf("LastPollTime: %v", err)
	}

	if !last.Equal(ts2) {
		t.Errorf("expected last poll time %v, got %v", ts2, last)
	}
}

func TestControllerMeta(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	sys := apex.SystemInfo{
		Hostname: "apex",
		Software: "5.10_8C18",
		Hardware: "1.0",
		Serial:   "AC5:12345",
		Type:     "A3",
		Timezone: "America/Chicago",
	}

	if err := db.WriteControllerMeta(ctx, ts, sys); err != nil {
		t.Fatalf("WriteControllerMeta: %v", err)
	}

	meta, err := db.ControllerMeta(ctx)
	if err != nil {
		t.Fatalf("ControllerMeta: %v", err)
	}

	if meta.Serial != "AC5:12345" {
		t.Errorf("expected serial AC5:12345, got %s", meta.Serial)
	}
	if meta.Hostname != "apex" {
		t.Errorf("expected hostname apex, got %s", meta.Hostname)
	}
	if meta.Software != "5.10_8C18" {
		t.Errorf("expected software 5.10_8C18, got %s", meta.Software)
	}
	if meta.Type != "A3" {
		t.Errorf("expected type A3, got %s", meta.Type)
	}
}
