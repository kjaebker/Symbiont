package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kjaebker/symbiont/internal/api"
	"github.com/kjaebker/symbiont/internal/db"
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
}

// Engine evaluates alert rules against current probe readings.
type Engine struct {
	sqlite      *db.SQLiteDB
	duck        *db.DuckDB
	notifier    notify.Notifier
	broadcaster *api.Broadcaster
	logger      *slog.Logger
	state       map[int64]*AlertState
	mu          sync.Mutex
}

// New creates a new alert evaluation engine.
func New(sqlite *db.SQLiteDB, duck *db.DuckDB, notifier notify.Notifier, broadcaster *api.Broadcaster, logger *slog.Logger) *Engine {
	return &Engine{
		sqlite:      sqlite,
		duck:        duck,
		notifier:    notifier,
		broadcaster: broadcaster,
		logger:      logger,
		state:       make(map[int64]*AlertState),
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

// Evaluate loads current probe readings and evaluates all enabled rules.
func (e *Engine) Evaluate(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.sqlite.ListEnabledAlertRules(ctx)
	if err != nil {
		e.logger.Error("alert engine: failed to load rules", "err", err)
		return
	}

	if len(rules) == 0 {
		return
	}

	readings, err := e.duck.CurrentProbeReadings(ctx)
	if err != nil {
		e.logger.Error("alert engine: failed to load probe readings", "err", err)
		return
	}

	// Build a map of probe name → latest value.
	probeValues := make(map[string]float64, len(readings))
	for _, r := range readings {
		probeValues[r.Name] = r.Value
	}

	for _, rule := range rules {
		value, exists := probeValues[rule.ProbeName]
		if !exists {
			// Probe doesn't exist in current data — skip silently.
			continue
		}

		breached := isBreached(rule, value)
		state := e.state[rule.ID]

		if breached {
			if state == nil || !state.Active {
				// New breach — fire the alert.
				e.fire(ctx, rule, value)
			} else {
				// Still breached — update peak and check cooldown for re-notify.
				e.updatePeak(ctx, rule, value)
			}
		} else if state != nil && state.Active {
			// Condition resolved — clear the alert.
			e.clear(ctx, rule)
		}
	}
}

// isBreached checks whether a probe value violates an alert rule's condition.
func isBreached(rule db.AlertRule, value float64) bool {
	switch rule.Condition {
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

func (e *Engine) fire(ctx context.Context, rule db.AlertRule, value float64) {
	now := time.Now()

	// Insert alert event in SQLite.
	eventID, err := e.sqlite.InsertAlertEvent(ctx, rule.ID, value)
	if err != nil {
		e.logger.Error("alert engine: failed to insert alert event", "err", err, "rule_id", rule.ID)
		return
	}

	e.state[rule.ID] = &AlertState{
		Active:    true,
		FiredAt:   now,
		PeakValue: value,
		EventID:   eventID,
	}

	// Send notification.
	e.sendNotification(ctx, rule, value)
	e.state[rule.ID].LastNotifiedAt = now

	// Publish SSE event.
	e.broadcaster.Publish(api.Event{
		Type: "alert_fired",
		Data: map[string]any{
			"rule_id":    rule.ID,
			"event_id":   eventID,
			"probe_name": rule.ProbeName,
			"severity":   rule.Severity,
			"value":      value,
			"condition":  rule.Condition,
			"fired_at":   now,
		},
	})

	e.logger.Warn("alert fired",
		"rule_id", rule.ID,
		"probe", rule.ProbeName,
		"severity", rule.Severity,
		"value", value,
		"condition", rule.Condition,
	)
}

func (e *Engine) clear(ctx context.Context, rule db.AlertRule) {
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

	// Publish SSE event.
	e.broadcaster.Publish(api.Event{
		Type: "alert_cleared",
		Data: map[string]any{
			"rule_id":    rule.ID,
			"event_id":   state.EventID,
			"probe_name": rule.ProbeName,
			"cleared_at": now,
		},
	})

	e.logger.Info("alert cleared",
		"rule_id", rule.ID,
		"probe", rule.ProbeName,
		"event_id", state.EventID,
	)
}

func (e *Engine) updatePeak(ctx context.Context, rule db.AlertRule, value float64) {
	state := e.state[rule.ID]
	if state == nil {
		return
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
		e.sendNotification(ctx, rule, value)
		state.LastNotifiedAt = time.Now()
	}
}

func (e *Engine) sendNotification(ctx context.Context, rule db.AlertRule, value float64) {
	if e.notifier == nil {
		return
	}

	title := formatAlertTitle(rule)
	body := formatAlertBody(rule, value)
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
	} else {
		_ = e.sqlite.MarkAlertEventNotified(ctx, e.state[rule.ID].EventID)
	}
}

func formatAlertTitle(rule db.AlertRule) string {
	icon := "⚠️"
	if rule.Severity == "critical" {
		icon = "🚨"
	}
	return fmt.Sprintf("%s %s %s", icon, rule.ProbeName, rule.Severity)
}

func formatAlertBody(rule db.AlertRule, value float64) string {
	switch rule.Condition {
	case "above":
		return fmt.Sprintf("%s is %.2f (threshold: above %.2f). Check your tank.", rule.ProbeName, value, *rule.ThresholdHigh)
	case "below":
		return fmt.Sprintf("%s is %.2f (threshold: below %.2f). Check your tank.", rule.ProbeName, value, *rule.ThresholdLow)
	case "outside_range":
		return fmt.Sprintf("%s is %.2f (range: %.2f–%.2f). Check your tank.", rule.ProbeName, value, *rule.ThresholdLow, *rule.ThresholdHigh)
	default:
		return fmt.Sprintf("%s is %.2f. Check your tank.", rule.ProbeName, value)
	}
}
