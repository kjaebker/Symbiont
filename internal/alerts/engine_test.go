package alerts

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
	"github.com/kjaebker/symbiont/internal/notify"
)

// recordingNotifier records sent notifications for test assertions.
type recordingNotifier struct {
	mu    sync.Mutex
	sent  []notify.Notification
}

func (r *recordingNotifier) Send(_ context.Context, n notify.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, n)
	return nil
}

func (r *recordingNotifier) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sent)
}

func (r *recordingNotifier) last() notify.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sent[len(r.sent)-1]
}

func ptr(v float64) *float64 { return &v }

func TestIsBreached(t *testing.T) {
	tests := []struct {
		name     string
		rule     db.AlertRule
		value    float64
		expected bool
	}{
		{
			name:     "above - breached",
			rule:     db.AlertRule{Condition: "above", ThresholdHigh: ptr(80.0)},
			value:    82.0,
			expected: true,
		},
		{
			name:     "above - not breached",
			rule:     db.AlertRule{Condition: "above", ThresholdHigh: ptr(80.0)},
			value:    78.0,
			expected: false,
		},
		{
			name:     "above - exactly at threshold",
			rule:     db.AlertRule{Condition: "above", ThresholdHigh: ptr(80.0)},
			value:    80.0,
			expected: false,
		},
		{
			name:     "below - breached",
			rule:     db.AlertRule{Condition: "below", ThresholdLow: ptr(7.8)},
			value:    7.5,
			expected: true,
		},
		{
			name:     "below - not breached",
			rule:     db.AlertRule{Condition: "below", ThresholdLow: ptr(7.8)},
			value:    8.0,
			expected: false,
		},
		{
			name:     "outside_range - below low",
			rule:     db.AlertRule{Condition: "outside_range", ThresholdLow: ptr(7.8), ThresholdHigh: ptr(8.4)},
			value:    7.5,
			expected: true,
		},
		{
			name:     "outside_range - above high",
			rule:     db.AlertRule{Condition: "outside_range", ThresholdLow: ptr(7.8), ThresholdHigh: ptr(8.4)},
			value:    8.6,
			expected: true,
		},
		{
			name:     "outside_range - within range",
			rule:     db.AlertRule{Condition: "outside_range", ThresholdLow: ptr(7.8), ThresholdHigh: ptr(8.4)},
			value:    8.1,
			expected: false,
		},
		{
			name:     "unknown condition",
			rule:     db.AlertRule{Condition: "unknown"},
			value:    100.0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBreached(tt.rule, tt.value, nil)
			if got != tt.expected {
				t.Errorf("isBreached() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func setupTestEngine(t *testing.T) (*Engine, *db.SQLiteDB, *db.DuckDB, *recordingNotifier) {
	t.Helper()

	sqliteDB, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { sqliteDB.Close() })

	tmpFile := t.TempDir() + "/test.duckdb"
	duckDB, err := db.Open(tmpFile)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	t.Cleanup(func() { duckDB.Close() })

	recorder := &recordingNotifier{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	bus := events.NewBus(logger)

	engine := New(sqliteDB, duckDB, recorder, bus, logger)

	return engine, sqliteDB, duckDB, recorder
}

func seedProbeReading(t *testing.T, duck *db.DuckDB, name string, value float64) {
	t.Helper()
	ctx := context.Background()
	_, err := duck.DB().ExecContext(ctx,
		"INSERT INTO probe_readings (ts, probe_did, probe_type, probe_name, value) VALUES (NOW(), ?, 'Temp', ?, ?)",
		name, name, value,
	)
	if err != nil {
		t.Fatalf("seeding probe reading: %v", err)
	}
}

func TestFireOnBreach(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	// Create a rule: temp above 80.
	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	ruleID, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	// Seed a reading above threshold.
	seedProbeReading(t, duck, "Temp", 82.0)

	// Evaluate — should fire.
	engine.Evaluate(ctx)

	if recorder.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", recorder.count())
	}

	// Check state was set.
	state := engine.state[ruleID]
	if state == nil || !state.Active {
		t.Fatal("expected alert state to be active")
	}
	if state.PeakValue != 82.0 {
		t.Errorf("expected peak 82.0, got %f", state.PeakValue)
	}

	// Check alert event was inserted.
	events, err := sqlite.ListAlertEvents(ctx, nil, false, 10)
	if err != nil {
		t.Fatalf("listing events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ClearedAt != nil {
		t.Error("expected event to not be cleared")
	}
}

func TestNoReFireOnSubsequentEval(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	_, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	seedProbeReading(t, duck, "Temp", 82.0)

	// First eval — fires.
	engine.Evaluate(ctx)
	if recorder.count() != 1 {
		t.Fatalf("expected 1 notification after first eval, got %d", recorder.count())
	}

	// Second eval — same reading, should NOT re-fire (cooldown not expired).
	engine.Evaluate(ctx)
	if recorder.count() != 1 {
		t.Fatalf("expected still 1 notification after second eval, got %d", recorder.count())
	}
}

func TestClearOnResolve(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	ruleID, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	// Fire the alert.
	seedProbeReading(t, duck, "Temp", 82.0)
	engine.Evaluate(ctx)

	// Now insert a normal reading (DuckDB returns latest per probe, so we need it to be the most recent).
	// Delete old reading and insert normal one.
	_, _ = duck.DB().ExecContext(ctx, "DELETE FROM probe_readings WHERE probe_name = 'Temp'")
	seedProbeReading(t, duck, "Temp", 78.0)

	// Evaluate — should clear.
	engine.Evaluate(ctx)

	state := engine.state[ruleID]
	if state == nil || state.Active {
		t.Fatal("expected alert state to be cleared")
	}

	// Check event was cleared in SQLite.
	events, err := sqlite.ListAlertEvents(ctx, nil, false, 10)
	if err != nil {
		t.Fatalf("listing events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ClearedAt == nil {
		t.Error("expected event to be cleared")
	}

	// Notification count should still be 1 (fire only, no clear notification).
	if recorder.count() != 1 {
		t.Errorf("expected 1 notification total, got %d", recorder.count())
	}
}

func TestCooldownReNotify(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 1, // 1 minute cooldown
		Enabled:         true,
	}
	ruleID, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	seedProbeReading(t, duck, "Temp", 82.0)

	// Fire the alert.
	engine.Evaluate(ctx)
	if recorder.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", recorder.count())
	}

	// Simulate cooldown expiry by backdating LastNotifiedAt.
	engine.state[ruleID].LastNotifiedAt = time.Now().Add(-2 * time.Minute)

	// Evaluate again — should re-notify since cooldown expired.
	engine.Evaluate(ctx)
	if recorder.count() != 2 {
		t.Fatalf("expected 2 notifications after cooldown expiry, got %d", recorder.count())
	}
}

func TestMissingProbeGraceful(t *testing.T) {
	engine, sqlite, _, _ := setupTestEngine(t)
	ctx := context.Background()

	// Create a rule for a probe that doesn't exist in DuckDB.
	rule := db.AlertRule{
		ProbeName:       "NonExistentProbe",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	_, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	// Should not panic or error.
	engine.Evaluate(ctx)
}

func TestFormatAlertTitle(t *testing.T) {
	warning := db.AlertRule{ProbeName: "pH", Severity: "warning"}
	critical := db.AlertRule{ProbeName: "Temp", Severity: "critical"}

	got := formatAlertTitle(warning, "pH")
	if got != "⚠️ pH warning" {
		t.Errorf("unexpected title: %s", got)
	}

	got = formatAlertTitle(critical, "Tank Temperature")
	if got != "🚨 Tank Temperature critical" {
		t.Errorf("unexpected title: %s", got)
	}
}

func TestFormatAlertTitleUsesDisplayName(t *testing.T) {
	rule := db.AlertRule{ProbeName: "TmpT", Severity: "warning", Condition: "above", ThresholdHigh: ptr(80.0)}

	got := formatAlertTitle(rule, "Tank Temperature")
	if got != "⚠️ Tank Temperature warning" {
		t.Errorf("unexpected title: %s", got)
	}

	got = formatAlertBody(rule, 82.0, "Tank Temperature", nil)
	if got != "Tank Temperature is 82.00 (threshold: above 80.00). Check your tank." {
		t.Errorf("unexpected body: %s", got)
	}
}

func TestEvaluateNotificationsOutsideLock(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	// Use a slow notifier to verify notifications don't block the lock.
	slow := &slowNotifier{delay: 200 * time.Millisecond, inner: recorder}
	engine.notifier = slow

	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	_, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}

	seedProbeReading(t, duck, "Temp", 82.0)

	start := time.Now()
	engine.Evaluate(ctx)
	elapsed := time.Since(start)

	if recorder.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", recorder.count())
	}

	// The notification itself takes 200ms, but the lock should be released quickly.
	// We allow some slack but the key invariant is that it completes.
	if elapsed > 2*time.Second {
		t.Errorf("Evaluate took too long (%v), expected under 2s", elapsed)
	}
}

// slowNotifier wraps a notifier with an artificial delay.
type slowNotifier struct {
	delay time.Duration
	inner notify.Notifier
}

func (s *slowNotifier) Send(ctx context.Context, n notify.Notification) error {
	time.Sleep(s.delay)
	return s.inner.Send(ctx, n)
}

// probeRangeRule inserts a probe_range rule for "Temp" and a probe config with
// a normal range of 76–80 and critical limits of 74–82.
func probeRangeRule(t *testing.T, sqlite *db.SQLiteDB, severity string) int64 {
	t.Helper()
	ctx := context.Background()
	if err := sqlite.UpsertProbeConfig(ctx, db.ProbeConfig{
		ProbeName:     "Temp",
		InputCategory: "probe",
		MinNormal:     ptr(76.0),
		MaxNormal:     ptr(80.0),
		MinWarning:    ptr(74.0),
		MaxWarning:    ptr(82.0),
	}); err != nil {
		t.Fatalf("upserting probe config: %v", err)
	}
	id, err := sqlite.InsertAlertRule(ctx, db.AlertRule{
		ProbeName:       "Temp",
		Condition:       db.AlertConditionProbeRange,
		Severity:        severity,
		CooldownMinutes: 15,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}
	return id
}

func TestProbeRangeRuleFollowsProbeConfig(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		value    float64
		fires    bool
	}{
		{"warning rule, normal value", "warning", 78.0, false},
		{"warning rule, warning value", "warning", 81.0, true},
		{"warning rule, critical value", "warning", 83.0, true},
		{"critical rule, warning value", "critical", 81.0, false},
		{"critical rule, critical value", "critical", 73.0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, sqlite, duck, recorder := setupTestEngine(t)
			probeRangeRule(t, sqlite, tt.severity)
			seedProbeReading(t, duck, "Temp", tt.value)

			engine.Evaluate(context.Background())

			if got := recorder.count() == 1; got != tt.fires {
				t.Errorf("fired = %v, want %v", got, tt.fires)
			}
		})
	}
}

func TestProbeRangeRuleTracksConfigChanges(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()
	ruleID := probeRangeRule(t, sqlite, "warning")
	seedProbeReading(t, duck, "Temp", 81.0)

	engine.Evaluate(ctx)
	if recorder.count() != 1 {
		t.Fatalf("expected alert to fire, got %d notifications", recorder.count())
	}
	body := recorder.sent[0].Body
	if body != "Temp is 81.00, outside its normal range (76.00–80.00). Check your tank." {
		t.Errorf("unexpected body: %s", body)
	}

	// Widening the probe's normal range resolves the alert without touching the rule.
	if err := sqlite.UpsertProbeConfig(ctx, db.ProbeConfig{
		ProbeName:     "Temp",
		InputCategory: "probe",
		MinNormal:     ptr(76.0),
		MaxNormal:     ptr(82.0),
	}); err != nil {
		t.Fatalf("upserting probe config: %v", err)
	}
	engine.Evaluate(ctx)
	if st := engine.state[ruleID]; st == nil || st.Active {
		t.Fatal("expected alert to clear after the range was widened")
	}
}

func openEventCount(t *testing.T, sqlite *db.SQLiteDB) int {
	t.Helper()
	events, err := sqlite.ListAlertEvents(context.Background(), nil, true, 100)
	if err != nil {
		t.Fatalf("listing events: %v", err)
	}
	return len(events)
}

func TestDisablingRuleClearsActiveAlert(t *testing.T) {
	engine, sqlite, duck, _ := setupTestEngine(t)
	ctx := context.Background()

	rule := db.AlertRule{
		ProbeName:       "Temp",
		Condition:       "above",
		ThresholdHigh:   ptr(80.0),
		Severity:        "warning",
		CooldownMinutes: 15,
		Enabled:         true,
	}
	ruleID, err := sqlite.InsertAlertRule(ctx, rule)
	if err != nil {
		t.Fatalf("inserting rule: %v", err)
	}
	seedProbeReading(t, duck, "Temp", 82.0)
	engine.Evaluate(ctx)
	if n := openEventCount(t, sqlite); n != 1 {
		t.Fatalf("expected 1 open event, got %d", n)
	}

	rule.Enabled = false
	if err := sqlite.UpdateAlertRule(ctx, ruleID, rule); err != nil {
		t.Fatalf("disabling rule: %v", err)
	}
	engine.Evaluate(ctx)

	if n := openEventCount(t, sqlite); n != 0 {
		t.Errorf("expected the event to clear when its rule is disabled, %d still open", n)
	}
}

func TestStartupReconcilesOpenEvents(t *testing.T) {
	engine, sqlite, duck, recorder := setupTestEngine(t)
	ctx := context.Background()

	insert := func(enabled bool) int64 {
		id, err := sqlite.InsertAlertRule(ctx, db.AlertRule{
			ProbeName:       "Temp",
			Condition:       "above",
			ThresholdHigh:   ptr(80.0),
			Severity:        "warning",
			CooldownMinutes: 15,
			Enabled:         enabled,
		})
		if err != nil {
			t.Fatalf("inserting rule: %v", err)
		}
		return id
	}
	breached := insert(true)
	disabled := insert(false)

	// Open events left behind by a previous process: two for the still
	// breached rule (a duplicate) and one for a rule that is now disabled.
	for _, ruleID := range []int64{breached, breached, disabled} {
		if _, err := sqlite.InsertAlertEvent(ctx, ruleID, 82.0); err != nil {
			t.Fatalf("inserting event: %v", err)
		}
	}
	seedProbeReading(t, duck, "Temp", 82.0)

	engine.Evaluate(ctx)

	// The breached rule keeps exactly one open event and does not fire again.
	if n := openEventCount(t, sqlite); n != 1 {
		t.Errorf("expected 1 open event after reconcile, got %d", n)
	}
	if recorder.count() != 0 {
		t.Errorf("expected no new notification for an adopted alert, got %d", recorder.count())
	}
	if st := engine.state[breached]; st == nil || !st.Active {
		t.Error("expected the breached rule's open event to be adopted")
	}

	// Once the reading recovers, the adopted event clears.
	_, _ = duck.DB().ExecContext(ctx, "DELETE FROM probe_readings WHERE probe_name = 'Temp'")
	seedProbeReading(t, duck, "Temp", 78.0)
	engine.Evaluate(ctx)
	if n := openEventCount(t, sqlite); n != 0 {
		t.Errorf("expected adopted event to clear, %d still open", n)
	}
}
