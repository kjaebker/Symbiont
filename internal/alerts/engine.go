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

// pendingNotification holds data for a notification that must be sent outside the lock.
type pendingNotification struct {
	rule        db.AlertRule
	value       float64
	displayName string
	eventID     int64
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

	// Build display name map from probe configs.
	displayNames := e.loadProbeDisplayNames(ctx)

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

		dn := displayNames[rule.ProbeName]
		if dn == "" {
			dn = splitCamelCase(rule.ProbeName)
		}

		breached := isBreached(rule, value)
		state := e.state[rule.ID]

		if breached {
			if state == nil || !state.Active {
				// New breach — fire the alert.
				p := e.fire(ctx, rule, value, dn)
				if p != nil {
					pending = append(pending, *p)
				}
			} else {
				// Still breached — update peak and check cooldown for re-notify.
				if p := e.updatePeak(ctx, rule, value, dn); p != nil {
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
		e.sendNotification(ctx, p.rule, p.value, p.displayName)
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

func (e *Engine) fire(ctx context.Context, rule db.AlertRule, value float64, displayName string) *pendingNotification {
	now := time.Now()

	// Insert alert event in SQLite.
	eventID, err := e.sqlite.InsertAlertEvent(ctx, rule.ID, value)
	if err != nil {
		e.logger.Error("alert engine: failed to insert alert event", "err", err, "rule_id", rule.ID)
		return nil
	}

	e.state[rule.ID] = &AlertState{
		Active:    true,
		FiredAt:   now,
		PeakValue: value,
		EventID:   eventID,
	}

	// Publish SSE event.
	e.broadcaster.Publish(api.Event{
		Type: "alert_fired",
		Data: map[string]any{
			"rule_id":      rule.ID,
			"event_id":     eventID,
			"probe_name":   rule.ProbeName,
			"display_name": displayName,
			"severity":     rule.Severity,
			"value":        value,
			"condition":    rule.Condition,
			"fired_at":     now,
		},
	})

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

	// Publish SSE event.
	e.broadcaster.Publish(api.Event{
		Type: "alert_cleared",
		Data: map[string]any{
			"rule_id":      rule.ID,
			"event_id":     state.EventID,
			"probe_name":   rule.ProbeName,
			"display_name": displayName,
			"cleared_at":   now,
		},
	})

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

func (e *Engine) sendNotification(ctx context.Context, rule db.AlertRule, value float64, displayName string) {
	if e.notifier == nil {
		return
	}

	title := formatAlertTitle(rule, displayName)
	body := formatAlertBody(rule, value, displayName)
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

func formatAlertBody(rule db.AlertRule, value float64, displayName string) string {
	switch rule.Condition {
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

// loadProbeDisplayNames loads probe configs and returns a map of probe name → display name.
func (e *Engine) loadProbeDisplayNames(ctx context.Context) map[string]string {
	configs, err := e.sqlite.ListProbeConfigs(ctx)
	if err != nil {
		e.logger.Error("alert engine: failed to load probe configs", "err", err)
		return nil
	}
	m := make(map[string]string, len(configs))
	for _, c := range configs {
		if c.DisplayName != nil {
			m[c.ProbeName] = *c.DisplayName
		}
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
