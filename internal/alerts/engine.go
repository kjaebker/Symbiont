package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
	"github.com/kjaebker/symbiont/internal/notify"
)

// AlertState tracks the in-memory state for an active alert.
type AlertState struct {
	Active         bool
	FiredAt        time.Time
	ClearedAt      time.Time
	PeakValue      float64
	EventID        int64     // SQLite alert_events.id for the current open event
	LastNotifiedAt time.Time // For cooldown enforcement
	ProbeName      string    // Kept so the alert can be cleared after its rule is gone
	DisplayName    string
}

// Engine evaluates alert rules against current probe readings.
type Engine struct {
	sqlite   *db.SQLiteDB
	duck     *db.DuckDB
	notifier notify.Notifier
	bus      *events.Bus
	logger   *slog.Logger
	state    map[int64]*AlertState
	mu       sync.Mutex

	// reconciled is set once the open events left in SQLite by a previous
	// process have been adopted or closed (see reconcile).
	reconciled bool
}

// New creates a new alert evaluation engine. bus is used to publish
// EvtAlertFired and EvtAlertCleared events; pass nil to disable publishing.
func New(sqlite *db.SQLiteDB, duck *db.DuckDB, notifier notify.Notifier, bus *events.Bus, logger *slog.Logger) *Engine {
	return &Engine{
		sqlite:   sqlite,
		duck:     duck,
		notifier: notifier,
		bus:      bus,
		logger:   logger,
		state:    make(map[int64]*AlertState),
	}
}

// Start runs the alert evaluation loop on a 10-second ticker.
func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	e.logger.Info("alert engine started")

	// Run an initial evaluation immediately.
	e.Evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("alert engine stopped")
			return
		case <-ticker.C:
			e.Evaluate(ctx)
		}
	}
}

// pendingNotification holds data for a notification that must be sent outside the lock.
type pendingNotification struct {
	rule        db.AlertRule
	value       float64
	displayName string
	eventID     int64
	cfg         *db.ProbeConfig
}

// Evaluate loads current probe readings and evaluates all enabled rules.
func (e *Engine) Evaluate(ctx context.Context) {
	e.mu.Lock()

	rules, err := e.sqlite.ListEnabledAlertRules(ctx)
	if err != nil {
		e.mu.Unlock()
		e.logger.Error("alert engine: failed to load rules", "err", err)
		return
	}

	enabled := make(map[int64]bool, len(rules))
	for _, r := range rules {
		enabled[r.ID] = true
	}

	if !e.reconciled {
		e.reconcile(ctx, enabled)
	}

	// An alert whose rule was disabled or deleted can never resolve through
	// evaluation, so close it here instead of leaving it active forever.
	for id, st := range e.state {
		if enabled[id] {
			continue
		}
		if st.Active {
			e.clear(ctx, db.AlertRule{ID: id, ProbeName: st.ProbeName}, st.DisplayName)
		}
		delete(e.state, id)
	}

	if len(rules) == 0 {
		e.mu.Unlock()
		return
	}

	readings, err := e.duck.CurrentProbeReadings(ctx)
	if err != nil {
		e.mu.Unlock()
		e.logger.Error("alert engine: failed to load probe readings", "err", err)
		return
	}

	// Probe configs supply display names and the ranges probe_range rules follow.
	configs := e.loadProbeConfigs(ctx)

	// Build a map of probe name → latest value.
	probeValues := make(map[string]float64, len(readings))
	for _, r := range readings {
		probeValues[r.Name] = r.Value
	}

	var pending []pendingNotification

	for _, rule := range rules {
		value, exists := probeValues[rule.ProbeName]
		if !exists {
			// Probe doesn't exist in current data — skip silently.
			continue
		}

		cfg := configs[rule.ProbeName]
		dn := ""
		if cfg != nil && cfg.DisplayName != nil {
			dn = *cfg.DisplayName
		}
		if dn == "" {
			dn = splitCamelCase(rule.ProbeName)
		}

		breached := isBreached(rule, value, cfg)
		state := e.state[rule.ID]

		if breached {
			if state == nil || !state.Active {
				// New breach — fire the alert.
				p := e.fire(ctx, rule, value, dn)
				if p != nil {
					p.cfg = cfg
					pending = append(pending, *p)
				}
			} else {
				// Still breached — update peak and check cooldown for re-notify.
				if p := e.updatePeak(ctx, rule, value, dn); p != nil {
					p.cfg = cfg
					pending = append(pending, *p)
				}
			}
		} else if state != nil && state.Active {
			// Condition resolved — clear the alert.
			e.clear(ctx, rule, dn)
		}
	}

	e.mu.Unlock()

	// Send notifications outside the lock to avoid blocking evaluation on I/O.
	for _, p := range pending {
		e.sendNotification(ctx, p.rule, p.value, p.displayName, p.cfg)
		e.mu.Lock()
		if s := e.state[p.rule.ID]; s != nil {
			s.LastNotifiedAt = time.Now()
		}
		e.mu.Unlock()
		if err := e.sqlite.MarkAlertEventNotified(ctx, p.eventID); err != nil {
			e.logger.Error("alert engine: failed to mark event notified", "err", err, "event_id", p.eventID)
		}
	}
}

// maxOpenEvents bounds how many open alert events reconcile reads.
const maxOpenEvents = 1000

// reconcile runs on the first evaluation after startup. The engine's state is
// in memory only, so alert events that were open when the previous process
// stopped would otherwise stay "active" forever, and a still-breached rule
// would fire a duplicate event. For each open event:
//   - its rule is enabled and has no adopted event yet: adopt it as the active
//     alert, so normal evaluation updates or clears it;
//   - otherwise (rule disabled, or an older duplicate): close it.
//
// Must be called with e.mu held. On a database error it leaves reconciled
// unset so the next tick retries.
func (e *Engine) reconcile(ctx context.Context, enabled map[int64]bool) {
	open, err := e.sqlite.ListAlertEvents(ctx, nil, true, maxOpenEvents)
	if err != nil {
		e.logger.Error("alert engine: failed to load open alert events", "err", err)
		return
	}

	now := time.Now()
	// ListAlertEvents returns newest first, so the first event seen for a rule
	// is the one to adopt.
	for _, ev := range open {
		if _, tracked := e.state[ev.RuleID]; enabled[ev.RuleID] && !tracked {
			st := &AlertState{
				Active:    true,
				FiredAt:   ev.FiredAt,
				PeakValue: ev.PeakValue,
				EventID:   ev.ID,
				// Don't re-notify just because the server restarted.
				LastNotifiedAt: now,
			}
			if ev.ProbeName != nil {
				st.ProbeName = *ev.ProbeName
				st.DisplayName = splitCamelCase(*ev.ProbeName)
			}
			e.state[ev.RuleID] = st
			continue
		}
		if err := e.sqlite.ClearAlertEvent(ctx, ev.ID); err != nil {
			e.logger.Error("alert engine: failed to clear stale alert event", "err", err, "event_id", ev.ID)
			continue
		}
		e.logger.Info("alert engine: cleared stale alert event", "event_id", ev.ID, "rule_id", ev.RuleID)
	}
	e.reconciled = true
}

// isBreached checks whether a probe value violates an alert rule's condition.
// cfg is the probe's config; only probe_range rules use it.
func isBreached(rule db.AlertRule, value float64, cfg *db.ProbeConfig) bool {
	switch rule.Condition {
	case db.AlertConditionProbeRange:
		// Fires when the probe's status is at least as bad as the rule's
		// severity: a warning rule also fires on critical.
		return db.ProbeStatusRank(db.ProbeStatus(value, cfg)) >= db.ProbeStatusRank(rule.Severity)
	case "above":
		return rule.ThresholdHigh != nil && value > *rule.ThresholdHigh
	case "below":
		return rule.ThresholdLow != nil && value < *rule.ThresholdLow
	case "outside_range":
		lowBreached := rule.ThresholdLow != nil && value < *rule.ThresholdLow
		highBreached := rule.ThresholdHigh != nil && value > *rule.ThresholdHigh
		return lowBreached || highBreached
	default:
		return false
	}
}

func (e *Engine) fire(ctx context.Context, rule db.AlertRule, value float64, displayName string) *pendingNotification {
	now := time.Now()

	// Insert alert event in SQLite.
	eventID, err := e.sqlite.InsertAlertEvent(ctx, rule.ID, value)
	if err != nil {
		e.logger.Error("alert engine: failed to insert alert event", "err", err, "rule_id", rule.ID)
		return nil
	}

	e.state[rule.ID] = &AlertState{
		Active:      true,
		FiredAt:     now,
		PeakValue:   value,
		EventID:     eventID,
		ProbeName:   rule.ProbeName,
		DisplayName: displayName,
	}

	// Publish via event bus (SSE, audit, notifier all subscribe).
	threshold := 0.0
	if rule.ThresholdHigh != nil {
		threshold = *rule.ThresholdHigh
	} else if rule.ThresholdLow != nil {
		threshold = *rule.ThresholdLow
	}
	if e.bus != nil {
		e.bus.Publish(events.NewAlertFired(
			rule.ID, displayName, rule.ProbeName,
			value, threshold, rule.Severity, rule.Condition, eventID,
		))
	}

	e.logger.Warn("alert fired",
		"rule_id", rule.ID,
		"probe", rule.ProbeName,
		"display_name", displayName,
		"severity", rule.Severity,
		"value", value,
		"condition", rule.Condition,
	)

	return &pendingNotification{rule: rule, value: value, displayName: displayName, eventID: eventID}
}

func (e *Engine) clear(ctx context.Context, rule db.AlertRule, displayName string) {
	state := e.state[rule.ID]
	if state == nil {
		return
	}

	now := time.Now()
	state.Active = false
	state.ClearedAt = now

	// Update alert event in SQLite.
	if err := e.sqlite.ClearAlertEvent(ctx, state.EventID); err != nil {
		e.logger.Error("alert engine: failed to clear alert event", "err", err, "event_id", state.EventID)
	}

	// Publish via event bus.
	if e.bus != nil {
		e.bus.Publish(events.NewAlertCleared(rule.ID, displayName, rule.ProbeName, state.EventID))
	}

	e.logger.Info("alert cleared",
		"rule_id", rule.ID,
		"probe", rule.ProbeName,
		"event_id", state.EventID,
	)
}

func (e *Engine) updatePeak(ctx context.Context, rule db.AlertRule, value float64, displayName string) *pendingNotification {
	state := e.state[rule.ID]
	if state == nil {
		return nil
	}

	// Update peak value if higher.
	if value > state.PeakValue {
		state.PeakValue = value
		if err := e.sqlite.UpdateAlertEventPeak(ctx, state.EventID, value); err != nil {
			e.logger.Error("alert engine: failed to update peak value", "err", err, "event_id", state.EventID)
		}
	}

	// Re-notify if cooldown has expired and still breached.
	cooldown := time.Duration(rule.CooldownMinutes) * time.Minute
	if cooldown > 0 && time.Since(state.LastNotifiedAt) >= cooldown {
		return &pendingNotification{rule: rule, value: value, displayName: displayName, eventID: state.EventID}
	}
	return nil
}

func (e *Engine) sendNotification(ctx context.Context, rule db.AlertRule, value float64, displayName string, cfg *db.ProbeConfig) {
	if e.notifier == nil {
		return
	}

	title := formatAlertTitle(rule, displayName)
	body := formatAlertBody(rule, value, displayName, cfg)
	priority := "high"
	if rule.Severity == "critical" {
		priority = "urgent"
	}

	n := notify.Notification{
		Title:    title,
		Body:     body,
		Priority: priority,
		Tags:     []string{rule.Severity, rule.ProbeName},
	}

	if err := e.notifier.Send(ctx, n); err != nil {
		e.logger.Error("alert engine: notification delivery failed",
			"err", err,
			"rule_id", rule.ID,
			"probe", rule.ProbeName,
		)
	}
}

func formatAlertTitle(rule db.AlertRule, displayName string) string {
	icon := "⚠️"
	if rule.Severity == "critical" {
		icon = "🚨"
	}
	return fmt.Sprintf("%s %s %s", icon, displayName, rule.Severity)
}

func formatAlertBody(rule db.AlertRule, value float64, displayName string, cfg *db.ProbeConfig) string {
	switch rule.Condition {
	case db.AlertConditionProbeRange:
		return formatProbeRangeBody(value, displayName, cfg)
	case "above":
		return fmt.Sprintf("%s is %.2f (threshold: above %.2f). Check your tank.", displayName, value, *rule.ThresholdHigh)
	case "below":
		return fmt.Sprintf("%s is %.2f (threshold: below %.2f). Check your tank.", displayName, value, *rule.ThresholdLow)
	case "outside_range":
		return fmt.Sprintf("%s is %.2f (range: %.2f–%.2f). Check your tank.", displayName, value, *rule.ThresholdLow, *rule.ThresholdHigh)
	default:
		return fmt.Sprintf("%s is %.2f. Check your tank.", displayName, value)
	}
}

// formatProbeRangeBody describes a probe_range breach using the range the
// probe actually left: the critical limits when its status is critical,
// otherwise the normal range.
func formatProbeRangeBody(value float64, displayName string, cfg *db.ProbeConfig) string {
	if cfg != nil && cfg.IsBinary {
		label := cfg.OnLabel
		if value == 0 {
			label = cfg.OffLabel
		}
		if label != nil && *label != "" {
			return fmt.Sprintf("%s is %s. Check your tank.", displayName, *label)
		}
		return fmt.Sprintf("%s is not in its OK state. Check your tank.", displayName)
	}

	status := db.ProbeStatus(value, cfg)
	low, high := cfg.RangeFor(status)
	kind := "normal range"
	if status == db.ProbeStatusCritical {
		kind = "critical limits"
	}
	switch {
	case low != nil && high != nil:
		return fmt.Sprintf("%s is %.2f, outside its %s (%.2f–%.2f). Check your tank.", displayName, value, kind, *low, *high)
	case low != nil:
		return fmt.Sprintf("%s is %.2f, outside its %s (minimum %.2f). Check your tank.", displayName, value, kind, *low)
	case high != nil:
		return fmt.Sprintf("%s is %.2f, outside its %s (maximum %.2f). Check your tank.", displayName, value, kind, *high)
	default:
		return fmt.Sprintf("%s is %.2f. Check your tank.", displayName, value)
	}
}

// loadProbeConfigs returns probe configs keyed by probe name.
func (e *Engine) loadProbeConfigs(ctx context.Context) map[string]*db.ProbeConfig {
	configs, err := e.sqlite.ListProbeConfigs(ctx)
	if err != nil {
		e.logger.Error("alert engine: failed to load probe configs", "err", err)
		return nil
	}
	m := make(map[string]*db.ProbeConfig, len(configs))
	for i := range configs {
		m[configs[i].ProbeName] = &configs[i]
	}
	return m
}

// splitCamelCase inserts spaces before uppercase letters in a CamelCase string.
func splitCamelCase(s string) string {
	if len(s) <= 1 {
		return s
	}
	var result []byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if i > 0 && ch >= 'A' && ch <= 'Z' {
			prevUpper := s[i-1] >= 'A' && s[i-1] <= 'Z'
			if !prevUpper {
				result = append(result, ' ')
			}
		}
		result = append(result, ch)
	}
	return string(result)
}
