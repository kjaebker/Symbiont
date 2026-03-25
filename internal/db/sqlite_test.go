package db

import (
	"context"
	"database/sql"
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
		ProbeName:   "Tmp",
		DisplayName: &displayName,
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
	events, err := db.ListOutletEvents(ctx, "", "", 50)
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
	events, err = db.ListOutletEvents(ctx, "base_Var1", "", 50)
	if err != nil {
		t.Fatalf("listing filtered events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(events))
	}

	// List filtered by non-existent outlet.
	events, err = db.ListOutletEvents(ctx, "nonexistent", "", 50)
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
		OutletID:    "base_Var1",
		DisplayName: &displayName,
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

func TestDeviceCRUD(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	deviceType := "heater"
	brand := "Eheim"
	d := Device{
		Name:       "Heater",
		DeviceType: &deviceType,
		Brand:      &brand,
		OutletID:   strPtr("base_Outlet1"),
	}

	// Insert.
	id, err := db.InsertDevice(ctx, d)
	if err != nil {
		t.Fatalf("inserting device: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero device ID")
	}

	// Get.
	got, err := db.GetDevice(ctx, id)
	if err != nil {
		t.Fatalf("getting device: %v", err)
	}
	if got.Name != "Heater" {
		t.Errorf("expected name 'Heater', got %q", got.Name)
	}
	if *got.DeviceType != "heater" {
		t.Errorf("expected device_type 'heater', got %q", *got.DeviceType)
	}
	if *got.Brand != "Eheim" {
		t.Errorf("expected brand 'Eheim', got %q", *got.Brand)
	}
	if len(got.ProbeNames) != 0 {
		t.Errorf("expected 0 probe names, got %d", len(got.ProbeNames))
	}

	// List.
	devices, err := db.ListDevices(ctx)
	if err != nil {
		t.Fatalf("listing devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	// Update.
	d.Name = "Main Heater"
	if err := db.UpdateDevice(ctx, id, d); err != nil {
		t.Fatalf("updating device: %v", err)
	}
	got, _ = db.GetDevice(ctx, id)
	if got.Name != "Main Heater" {
		t.Errorf("expected 'Main Heater', got %q", got.Name)
	}

	// Delete.
	if err := db.DeleteDevice(ctx, id); err != nil {
		t.Fatalf("deleting device: %v", err)
	}
	devices, _ = db.ListDevices(ctx)
	if len(devices) != 0 {
		t.Errorf("expected 0 devices after delete, got %d", len(devices))
	}

	// Delete non-existent.
	if err := db.DeleteDevice(ctx, 999); err == nil {
		t.Fatal("expected error deleting non-existent device")
	}
}

func TestDeviceProbeLink(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	// Create a device.
	id, err := db.InsertDevice(ctx, Device{Name: "Return Pump"})
	if err != nil {
		t.Fatalf("inserting device: %v", err)
	}

	// Link probes.
	if err := db.SetDeviceProbes(ctx, id, []string{"ReturnW", "ReturnA"}); err != nil {
		t.Fatalf("setting device probes: %v", err)
	}

	// Verify links.
	got, err := db.GetDevice(ctx, id)
	if err != nil {
		t.Fatalf("getting device: %v", err)
	}
	if len(got.ProbeNames) != 2 {
		t.Fatalf("expected 2 probe names, got %d", len(got.ProbeNames))
	}

	// GetDeviceByProbeName.
	d, err := db.GetDeviceByProbeName(ctx, "ReturnW")
	if err != nil {
		t.Fatalf("getting device by probe: %v", err)
	}
	if d == nil || d.ID != id {
		t.Errorf("expected device %d, got %v", id, d)
	}

	// Replace probes (remove ReturnA, keep ReturnW).
	if err := db.SetDeviceProbes(ctx, id, []string{"ReturnW"}); err != nil {
		t.Fatalf("replacing device probes: %v", err)
	}
	got, _ = db.GetDevice(ctx, id)
	if len(got.ProbeNames) != 1 {
		t.Errorf("expected 1 probe name after replace, got %d", len(got.ProbeNames))
	}

	// Delete device — probe should be unlinked.
	if err := db.DeleteDevice(ctx, id); err != nil {
		t.Fatalf("deleting device: %v", err)
	}
	cfg, err := db.GetProbeConfig(ctx, "ReturnW")
	if err != nil {
		t.Fatalf("getting probe config after device delete: %v", err)
	}
	if cfg.DeviceID != nil {
		t.Error("expected device_id to be NULL after device deletion")
	}
}

func TestSyncDeviceDisplayNames(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	// Create device with outlet.
	id, err := db.InsertDevice(ctx, Device{Name: "Heater", OutletID: strPtr("base_Outlet1")})
	if err != nil {
		t.Fatalf("inserting device: %v", err)
	}

	// Create outlet config.
	if err := db.UpsertOutletConfig(ctx, OutletConfig{OutletID: "base_Outlet1"}); err != nil {
		t.Fatalf("upserting outlet config: %v", err)
	}

	// Link probes.
	if err := db.SetDeviceProbes(ctx, id, []string{"HeatW", "HeatA", "HeatTemp"}); err != nil {
		t.Fatalf("setting device probes: %v", err)
	}

	// Sync names.
	if err := db.SyncDeviceDisplayNames(ctx, id, "Heater"); err != nil {
		t.Fatalf("syncing display names: %v", err)
	}

	// Verify outlet got bare name.
	oc, err := db.GetOutletConfig(ctx, "base_Outlet1")
	if err != nil {
		t.Fatalf("getting outlet config: %v", err)
	}
	if oc.DisplayName == nil || *oc.DisplayName != "Heater" {
		t.Errorf("expected outlet display name 'Heater', got %v", oc.DisplayName)
	}

	// Verify all probes got the bare device name.
	for _, probeName := range []string{"HeatW", "HeatA", "HeatTemp"} {
		pc, err := db.GetProbeConfig(ctx, probeName)
		if err != nil {
			t.Fatalf("getting probe config %s: %v", probeName, err)
		}
		if pc.DisplayName == nil || *pc.DisplayName != "Heater" {
			t.Errorf("expected probe %s display name 'Heater', got %v", probeName, pc.DisplayName)
		}
	}
}

func strPtr(s string) *string { return &s }

func TestDashboardItemsLifecycle(t *testing.T) {
	db := openTestSQLite(t)
	ctx := context.Background()

	// Initially empty.
	items, err := db.ListDashboardItems(ctx)
	if err != nil {
		t.Fatalf("listing dashboard items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}

	// Add items.
	probeName := "Tmp"
	id1, err := db.AddDashboardItem(ctx, DashboardItem{ItemType: "probe", ReferenceID: &probeName})
	if err != nil {
		t.Fatalf("adding probe item: %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero ID")
	}

	outletID := "base_Var1"
	id2, err := db.AddDashboardItem(ctx, DashboardItem{ItemType: "outlet", ReferenceID: &outletID})
	if err != nil {
		t.Fatalf("adding outlet item: %v", err)
	}

	items, _ = db.ListDashboardItems(ctx)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].SortOrder != 1 || items[1].SortOrder != 2 {
		t.Errorf("unexpected sort orders: %d, %d", items[0].SortOrder, items[1].SortOrder)
	}

	// Replace layout.
	sepLabel := "My Section"
	err = db.ReplaceDashboardLayout(ctx, []DashboardItem{
		{ItemType: "separator", Label: &sepLabel},
		{ItemType: "outlet", ReferenceID: &outletID},
		{ItemType: "probe", ReferenceID: &probeName},
	})
	if err != nil {
		t.Fatalf("replacing layout: %v", err)
	}

	items, _ = db.ListDashboardItems(ctx)
	if len(items) != 3 {
		t.Fatalf("expected 3 items after replace, got %d", len(items))
	}
	if items[0].ItemType != "separator" {
		t.Errorf("expected first item to be separator, got %q", items[0].ItemType)
	}

	// Remove by ID.
	if err := db.RemoveDashboardItem(ctx, items[2].ID); err != nil {
		t.Fatalf("removing item: %v", err)
	}
	items, _ = db.ListDashboardItems(ctx)
	if len(items) != 2 {
		t.Fatalf("expected 2 items after remove, got %d", len(items))
	}

	// Remove by ref.
	if err := db.RemoveDashboardItemByRef(ctx, "outlet", outletID); err != nil {
		t.Fatalf("removing by ref: %v", err)
	}
	items, _ = db.ListDashboardItems(ctx)
	if len(items) != 1 {
		t.Fatalf("expected 1 item after remove by ref, got %d", len(items))
	}

	// Remove non-existent — should error.
	if err := db.RemoveDashboardItem(ctx, 999); err == nil {
		t.Fatal("expected error removing non-existent item")
	}

	_ = id2
}

func TestMigrateDashboardLayout(t *testing.T) {
	// Use raw sql.Open to set up legacy data before OpenSQLite runs migration.
	rawDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening raw sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })

	if err := CreateSQLiteSchema(rawDB); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	ctx := context.Background()

	// Simulate an upgrade from the old schema by adding legacy columns.
	for _, alt := range []string{
		"ALTER TABLE probe_config ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE probe_config ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE outlet_config ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE outlet_config ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE devices ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE devices ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
	} {
		if _, err := rawDB.ExecContext(ctx, alt); err != nil {
			t.Fatalf("adding legacy column: %v", err)
		}
	}

	// Insert legacy probe configs with hidden/display_order.
	rawDB.ExecContext(ctx, "INSERT INTO probe_config (probe_name, display_order, hidden) VALUES ('Tmp', 1, 0)")
	rawDB.ExecContext(ctx, "INSERT INTO probe_config (probe_name, display_order, hidden) VALUES ('pH', 2, 0)")
	rawDB.ExecContext(ctx, "INSERT INTO probe_config (probe_name, display_order, hidden) VALUES ('HiddenProbe', 3, 1)")

	// Insert legacy outlet config.
	rawDB.ExecContext(ctx, "INSERT INTO outlet_config (outlet_id, display_order, hidden) VALUES ('base_Var1', 1, 0)")

	// Insert legacy device.
	rawDB.ExecContext(ctx, "INSERT INTO devices (name, display_order, hidden) VALUES ('Heater', 1, 0)")

	// Run migration.
	s := &SQLiteDB{db: rawDB, path: ":memory:"}
	if err := s.MigrateDashboardLayout(ctx); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	items, err := s.ListDashboardItems(ctx)
	if err != nil {
		t.Fatalf("listing items: %v", err)
	}

	// Expect: separator + 2 probes + separator + 1 device + separator + 1 outlet = 7
	if len(items) != 7 {
		t.Fatalf("expected 7 items, got %d", len(items))
	}
	if items[0].ItemType != "separator" {
		t.Errorf("expected first item to be separator, got %q", items[0].ItemType)
	}
	if items[1].ItemType != "probe" || *items[1].ReferenceID != "Tmp" {
		t.Errorf("expected second item to be probe Tmp, got %s/%v", items[1].ItemType, items[1].ReferenceID)
	}
}

func TestMigrateDashboardLayoutIdempotent(t *testing.T) {
	rawDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening raw sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })

	if err := CreateSQLiteSchema(rawDB); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	ctx := context.Background()

	// Simulate upgrade from old schema with legacy columns.
	for _, alt := range []string{
		"ALTER TABLE probe_config ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE probe_config ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE outlet_config ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE outlet_config ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE devices ADD COLUMN display_order INTEGER NOT NULL DEFAULT 999",
		"ALTER TABLE devices ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0",
	} {
		if _, err := rawDB.ExecContext(ctx, alt); err != nil {
			t.Fatalf("adding legacy column: %v", err)
		}
	}
	rawDB.ExecContext(ctx, "INSERT INTO probe_config (probe_name, display_order, hidden) VALUES ('Tmp', 1, 0)")

	s := &SQLiteDB{db: rawDB, path: ":memory:"}

	// First migration.
	if err := s.MigrateDashboardLayout(ctx); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	items1, _ := s.ListDashboardItems(ctx)

	// Second migration — should be no-op.
	if err := s.MigrateDashboardLayout(ctx); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	items2, _ := s.ListDashboardItems(ctx)

	if len(items1) != len(items2) {
		t.Errorf("expected same count after idempotent migration: %d vs %d", len(items1), len(items2))
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
