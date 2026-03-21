package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
)

func TestDeleteOldRows(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cleanup_test.db")

	duckDB, err := Open(dbPath)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { duckDB.Close() })

	ctx := context.Background()

	// Insert old data (400 days ago).
	oldTS := time.Now().Add(-400 * 24 * time.Hour)
	oldInputs := []apex.Input{
		{DID: "base_Temp", Name: "Tmp", Value: 78.0, Type: "Temp"},
	}
	oldOutputs := []apex.Output{
		{DID: "base_Var1", ID: 1, Name: "Return", Type: "outlet", Status: []string{"ON", "", "OK", ""}},
	}
	oldSystem := apex.SystemInfo{
		Serial: "AC5:12345", Hostname: "apex", Software: "5.10",
		Hardware: "1.0", Type: "AC5", Timezone: "UTC",
	}
	oldPower := apex.PowerInfo{Failed: oldTS.Unix(), Restored: oldTS.Add(time.Minute).Unix()}

	if err := duckDB.WriteProbeReadings(ctx, oldTS, oldInputs); err != nil {
		t.Fatalf("writing old probe readings: %v", err)
	}
	if err := duckDB.WriteOutletStates(ctx, oldTS, oldOutputs); err != nil {
		t.Fatalf("writing old outlet states: %v", err)
	}
	if err := duckDB.WritePowerEvents(ctx, oldTS, oldPower); err != nil {
		t.Fatalf("writing old power events: %v", err)
	}
	if err := duckDB.WriteControllerMeta(ctx, oldTS, oldSystem); err != nil {
		t.Fatalf("writing old controller meta: %v", err)
	}

	// Insert recent data (1 hour ago).
	recentTS := time.Now().Add(-1 * time.Hour)
	recentPower := apex.PowerInfo{Failed: recentTS.Unix(), Restored: recentTS.Add(time.Minute).Unix()}

	if err := duckDB.WriteProbeReadings(ctx, recentTS, oldInputs); err != nil {
		t.Fatalf("writing recent probe readings: %v", err)
	}
	if err := duckDB.WriteOutletStates(ctx, recentTS, oldOutputs); err != nil {
		t.Fatalf("writing recent outlet states: %v", err)
	}
	if err := duckDB.WritePowerEvents(ctx, recentTS, recentPower); err != nil {
		t.Fatalf("writing recent power events: %v", err)
	}
	if err := duckDB.WriteControllerMeta(ctx, recentTS, oldSystem); err != nil {
		t.Fatalf("writing recent controller meta: %v", err)
	}

	// Delete rows older than 365 days.
	result, err := duckDB.DeleteOldRows(ctx, 365)
	if err != nil {
		t.Fatalf("deleting old rows: %v", err)
	}

	if result.ProbeReadings != 1 {
		t.Errorf("expected 1 probe reading deleted, got %d", result.ProbeReadings)
	}
	if result.OutletStates != 1 {
		t.Errorf("expected 1 outlet state deleted, got %d", result.OutletStates)
	}
	if result.PowerEvents != 2 {
		t.Errorf("expected 2 power events deleted, got %d", result.PowerEvents)
	}
	if result.ControllerMeta != 1 {
		t.Errorf("expected 1 controller meta deleted, got %d", result.ControllerMeta)
	}

	// Verify recent rows still exist.
	readings, err := duckDB.CurrentProbeReadings(ctx)
	if err != nil {
		t.Fatalf("querying probe readings: %v", err)
	}
	if len(readings) != 1 {
		t.Errorf("expected 1 remaining probe reading, got %d", len(readings))
	}

	outlets, err := duckDB.CurrentOutletStates(ctx)
	if err != nil {
		t.Fatalf("querying outlet states: %v", err)
	}
	if len(outlets) != 1 {
		t.Errorf("expected 1 remaining outlet state, got %d", len(outlets))
	}
}
