package db

import (
	"context"
	"testing"
)

func openTestSQLite(t *testing.T) *SQLiteDB {
	t.Helper()
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteSchemaIdempotent(t *testing.T) {
	db := openTestSQLite(t)
	// Running schema creation again should not error.
	if err := CreateSQLiteSchema(db.DB()); err != nil {
		t.Fatalf("second schema creation failed: %v", err)
	}
}

func TestTokenLifecycle(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	// Insert a token.
	token, err := db.InsertToken(ctx, "test-token")
	if err != nil {
		t.Fatalf("inserting token: %v", err)
	}
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char token, got %d chars", len(token))
	}

	// Validate it.
	valid, id := db.ValidateToken(ctx, token)
	if !valid {
		t.Fatal("expected token to be valid")
	}
	if id == 0 {
		t.Fatal("expected non-zero token ID")
	}

	// Validate a bad token.
	valid, _ = db.ValidateToken(ctx, "not-a-real-token")
	if valid {
		t.Fatal("expected invalid token to fail validation")
	}

	// Touch it.
	if err := db.TouchToken(ctx, id); err != nil {
		t.Fatalf("touching token: %v", err)
	}

	// List tokens.
	tokens, err := db.ListTokens(ctx)
	if err != nil {
		t.Fatalf("listing tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].Label != "test-token" {
		t.Errorf("expected label 'test-token', got %q", tokens[0].Label)
	}
	if tokens[0].LastUsed == nil {
		t.Error("expected last_used to be set after touch")
	}

	// Delete it.
	if err := db.DeleteToken(ctx, id); err != nil {
		t.Fatalf("deleting token: %v", err)
	}

	// Validate again — should fail.
	valid, _ = db.ValidateToken(ctx, token)
	if valid {
		t.Fatal("expected deleted token to fail validation")
	}

	// Delete non-existent — should error.
	if err := db.DeleteToken(ctx, 999); err == nil {
		t.Fatal("expected error deleting non-existent token")
	}
}

func TestProbeConfigUpsert(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	displayName := "Temperature"
	cfg := ProbeConfig{
		ProbeName:    "Tmp",
		DisplayName:  &displayName,
		DisplayOrder: 1,
	}

	// Insert.
	if err := db.UpsertProbeConfig(ctx, cfg); err != nil {
		t.Fatalf("inserting probe config: %v", err)
	}

	// Read back.
	got, err := db.GetProbeConfig(ctx, "Tmp")
	if err != nil {
		t.Fatalf("getting probe config: %v", err)
	}
	if *got.DisplayName != "Temperature" {
		t.Errorf("expected display_name 'Temperature', got %q", *got.DisplayName)
	}

	// Update.
	newName := "Water Temp"
	cfg.DisplayName = &newName
	if err := db.UpsertProbeConfig(ctx, cfg); err != nil {
		t.Fatalf("updating probe config: %v", err)
	}

	got, err = db.GetProbeConfig(ctx, "Tmp")
	if err != nil {
		t.Fatalf("getting updated probe config: %v", err)
	}
	if *got.DisplayName != "Water Temp" {
		t.Errorf("expected display_name 'Water Temp', got %q", *got.DisplayName)
	}

	// List.
	configs, err := db.ListProbeConfigs(ctx)
	if err != nil {
		t.Fatalf("listing probe configs: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(configs))
	}
}

func TestOutletEventInsertAndList(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	fromState := "OFF"
	outletName := "Return Pump"
	event := OutletEvent{
		OutletID:    "base_Var1",
		OutletName:  &outletName,
		FromState:   &fromState,
		ToState:     "ON",
		InitiatedBy: "api",
	}

	if err := db.InsertOutletEvent(ctx, event); err != nil {
		t.Fatalf("inserting outlet event: %v", err)
	}

	// List all events.
	events, err := db.ListOutletEvents(ctx, "", 50)
	if err != nil {
		t.Fatalf("listing all outlet events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ToState != "ON" {
		t.Errorf("expected to_state 'ON', got %q", events[0].ToState)
	}

	// List filtered by outlet.
	events, err = db.ListOutletEvents(ctx, "base_Var1", 50)
	if err != nil {
		t.Fatalf("listing filtered events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(events))
	}

	// List filtered by non-existent outlet.
	events, err = db.ListOutletEvents(ctx, "nonexistent", 50)
	if err != nil {
		t.Fatalf("listing events for nonexistent outlet: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestAlertRuleCRUD(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	low := 76.0
	high := 82.0
	rule := AlertRule{
		ProbeName:       "Tmp",
		Condition:       "outside_range",
		ThresholdLow:    &low,
		ThresholdHigh:   &high,
		Severity:        "warning",
		CooldownMinutes: 30,
		Enabled:         true,
	}

	// Insert.
	id, err := db.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting alert rule: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero rule ID")
	}

	// List enabled.
	rules, err := db.ListEnabledAlertRules(ctx)
	if err != nil {
		t.Fatalf("listing enabled rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	// Update.
	rule.Severity = "critical"
	if err := db.UpdateAlertRule(ctx, id, rule); err != nil {
		t.Fatalf("updating alert rule: %v", err)
	}

	rules, err = db.ListAlertRules(ctx)
	if err != nil {
		t.Fatalf("listing all rules: %v", err)
	}
	if rules[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", rules[0].Severity)
	}

	// Delete.
	if err := db.DeleteAlertRule(ctx, id); err != nil {
		t.Fatalf("deleting alert rule: %v", err)
	}

	rules, err = db.ListAlertRules(ctx)
	if err != nil {
		t.Fatalf("listing after delete: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestOutletConfigUpsert(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	displayName := "Return Pump"
	cfg := OutletConfig{
		OutletID:     "base_Var1",
		DisplayName:  &displayName,
		DisplayOrder: 1,
	}

	if err := db.UpsertOutletConfig(ctx, cfg); err != nil {
		t.Fatalf("inserting outlet config: %v", err)
	}

	got, err := db.GetOutletConfig(ctx, "base_Var1")
	if err != nil {
		t.Fatalf("getting outlet config: %v", err)
	}
	if *got.DisplayName != "Return Pump" {
		t.Errorf("expected 'Return Pump', got %q", *got.DisplayName)
	}

	configs, err := db.ListOutletConfigs(ctx)
	if err != nil {
		t.Fatalf("listing outlet configs: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(configs))
	}
}

func TestBackupJobLifecycle(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	path := "/var/lib/symbiont/backups/2024-01-01.db"
	job := BackupJob{
		Status: "success",
		Path:   &path,
	}

	id, err := db.InsertBackupJob(ctx, job)
	if err != nil {
		t.Fatalf("inserting backup job: %v", err)
	}

	if err := db.UpdateBackupJob(ctx, id, "failed", "disk full"); err != nil {
		t.Fatalf("updating backup job: %v", err)
	}

	jobs, err := db.ListBackupJobs(ctx, 10)
	if err != nil {
		t.Fatalf("listing backup jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != "failed" {
		t.Errorf("expected status 'failed', got %q", jobs[0].Status)
	}
}

func TestEnsureDefaultToken(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	// First call should create a token.
	token, created, err := db.EnsureDefaultToken(ctx)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !created {
		t.Fatal("expected token to be created on first call")
	}
	if len(token) != 64 {
		t.Errorf("expected 64-char token, got %d chars", len(token))
	}

	// Second call should not create another token.
	token2, created2, err := db.EnsureDefaultToken(ctx)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created2 {
		t.Fatal("expected no token created on second call")
	}
	if token2 != "" {
		t.Errorf("expected empty token on second call, got %q", token2)
	}
}
