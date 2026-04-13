// Package events defines the typed system events that flow through the event bus.
// SystemEvent is a sealed interface — only types defined in this package can
// implement it, ensuring all event kinds are enumerable and auditable.
package events

import (
	"strconv"
	"time"
)

// Correlatable is an optional interface for events associated with a specific
// entity (outlet, livestock, alert rule, etc.). The audit subscriber uses it to
// populate the correlation_id column so events can be queried by entity.
type Correlatable interface {
	CorrelationID() string
}

// SystemEvent is the sealed interface for all system events.
// The unexported sealed() method prevents implementation outside this package.
type SystemEvent interface {
	Kind() string
	OccurredAt() time.Time
	sealed()
}

// base embeds common fields and satisfies the sealed() constraint.
type base struct {
	At time.Time `json:"at"`
}

func (b base) OccurredAt() time.Time { return b.At }
func (b base) sealed()               {}

// now returns the current time, used as a default for events that don't supply At.
func now() time.Time { return time.Now() }

// EvtFeedModeActivated is published when a Neptune Apex feed mode is activated.
type EvtFeedModeActivated struct {
	base
	Mode    string `json:"mode"`     // "A", "B", "C", or "D"
	FeedNum int    `json:"feed_num"` // 1–4
}

func (e EvtFeedModeActivated) Kind() string { return "feed_mode_activated" }

// NewFeedModeActivated constructs an EvtFeedModeActivated with At = now.
func NewFeedModeActivated(mode string, feedNum int) EvtFeedModeActivated {
	return EvtFeedModeActivated{base: base{At: now()}, Mode: mode, FeedNum: feedNum}
}

// EvtOutletChanged is published after a successful outlet state change.
type EvtOutletChanged struct {
	base
	OutletID    string `json:"outlet_id"`
	Name        string `json:"name"`
	PrevState   string `json:"prev_state"` // empty string if unknown
	NewState    string `json:"new_state"`
	InitiatedBy string `json:"initiated_by"` // "ui", "cli", "mcp", "api", "apex"
}

func (e EvtOutletChanged) Kind() string          { return "outlet_changed" }
func (e EvtOutletChanged) CorrelationID() string { return e.OutletID }

// NewOutletChanged constructs an EvtOutletChanged with At = now.
func NewOutletChanged(outletID, name, prevState, newState, initiatedBy string) EvtOutletChanged {
	return EvtOutletChanged{
		base:        base{At: now()},
		OutletID:    outletID,
		Name:        name,
		PrevState:   prevState,
		NewState:    newState,
		InitiatedBy: initiatedBy,
	}
}

// EvtAlertFired is published when an alert rule transitions to breached.
type EvtAlertFired struct {
	base
	RuleID    int64   `json:"rule_id"`
	RuleName  string  `json:"rule_name"` // probe display name or probe name
	ProbeName string  `json:"probe_name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"` // the relevant threshold that was exceeded
	Severity  string  `json:"severity"`
	Condition string  `json:"condition"`
	EventID   int64   `json:"event_id"` // alert_events.id
}

func (e EvtAlertFired) Kind() string          { return "alert_fired" }
func (e EvtAlertFired) CorrelationID() string { return strconv.FormatInt(e.RuleID, 10) }

// EvtAlertCleared is published when a previously fired alert resolves.
type EvtAlertCleared struct {
	base
	RuleID    int64  `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	ProbeName string `json:"probe_name"`
	EventID   int64  `json:"event_id"`
}

func (e EvtAlertCleared) Kind() string          { return "alert_cleared" }
func (e EvtAlertCleared) CorrelationID() string { return strconv.FormatInt(e.RuleID, 10) }

// EvtObservationRecorded is published when a livestock observation is created.
type EvtObservationRecorded struct {
	base
	ObservationID int64  `json:"observation_id"`
	LivestockID   int64  `json:"livestock_id"`
	LivestockName string `json:"livestock_name"`
	Status        string `json:"status"`      // may be empty if observation has no status
	PrevStatus    string `json:"prev_status"` // previous livestock status
	HasImage      bool   `json:"has_image"`
}

func (e EvtObservationRecorded) Kind() string          { return "observation_recorded" }
func (e EvtObservationRecorded) CorrelationID() string { return strconv.FormatInt(e.LivestockID, 10) }

// NewObservationRecorded constructs an EvtObservationRecorded with At = now.
func NewObservationRecorded(obsID, livestockID int64, livestockName, status, prevStatus string, hasImage bool) EvtObservationRecorded {
	return EvtObservationRecorded{
		base:          base{At: now()},
		ObservationID: obsID,
		LivestockID:   livestockID,
		LivestockName: livestockName,
		Status:        status,
		PrevStatus:    prevStatus,
		HasImage:      hasImage,
	}
}

// EvtLivestockAdded is published after a new livestock item is inserted.
type EvtLivestockAdded struct {
	base
	LivestockID int64  `json:"livestock_id"`
	Name        string `json:"name"`
	Species     string `json:"species"` // may be empty
	Type        string `json:"type"`    // fish, coral, invertebrate, other
}

func (e EvtLivestockAdded) Kind() string          { return "livestock_added" }
func (e EvtLivestockAdded) CorrelationID() string { return strconv.FormatInt(e.LivestockID, 10) }

// NewLivestockAdded constructs an EvtLivestockAdded with At = now.
func NewLivestockAdded(id int64, name, species, livestockType string) EvtLivestockAdded {
	return EvtLivestockAdded{base: base{At: now()}, LivestockID: id, Name: name, Species: species, Type: livestockType}
}

// EvtLivestockUpdated is published after a livestock item's fields change.
type EvtLivestockUpdated struct {
	base
	LivestockID   int64    `json:"livestock_id"`
	Name          string   `json:"name"`
	ChangedFields []string `json:"changed_fields"` // e.g. ["status", "quantity"]
}

func (e EvtLivestockUpdated) Kind() string          { return "livestock_updated" }
func (e EvtLivestockUpdated) CorrelationID() string { return strconv.FormatInt(e.LivestockID, 10) }

// NewLivestockUpdated constructs an EvtLivestockUpdated with At = now.
func NewLivestockUpdated(id int64, name string, changedFields []string) EvtLivestockUpdated {
	return EvtLivestockUpdated{base: base{At: now()}, LivestockID: id, Name: name, ChangedFields: changedFields}
}

// EvtJournalEntryCreated is published after a journal entry is inserted.
type EvtJournalEntryCreated struct {
	base
	EntryID  int64  `json:"entry_id"`
	Category string `json:"category"`
	Source   string `json:"source"` // "manual", "system", "ai"
}

func (e EvtJournalEntryCreated) Kind() string          { return "journal_entry_created" }
func (e EvtJournalEntryCreated) CorrelationID() string { return strconv.FormatInt(e.EntryID, 10) }

// NewJournalEntryCreated constructs an EvtJournalEntryCreated with At = now.
func NewJournalEntryCreated(entryID int64, category, source string) EvtJournalEntryCreated {
	return EvtJournalEntryCreated{base: base{At: now()}, EntryID: entryID, Category: category, Source: source}
}

// EvtPollCycleCompleted is published after each successful Apex poll cycle.
type EvtPollCycleCompleted struct {
	base
	DurationMs int64 `json:"duration_ms"`
	ProbeCount int   `json:"probe_count"`
}

func (e EvtPollCycleCompleted) Kind() string { return "poll_cycle_completed" }

// NewPollCycleCompleted constructs an EvtPollCycleCompleted with At = now.
func NewPollCycleCompleted(durationMs int64, probeCount int) EvtPollCycleCompleted {
	return EvtPollCycleCompleted{base: base{At: now()}, DurationMs: durationMs, ProbeCount: probeCount}
}

// EvtConfigChanged is published when a probe or outlet config is updated.
type EvtConfigChanged struct {
	base
	Key       string `json:"key"`        // e.g. "probe:Tmp" or "outlet:3_1"
	ChangedBy string `json:"changed_by"` // initiator description
}

func (e EvtConfigChanged) Kind() string { return "config_changed" }

// NewConfigChanged constructs an EvtConfigChanged with At = now.
func NewConfigChanged(key, changedBy string) EvtConfigChanged {
	return EvtConfigChanged{base: base{At: now()}, Key: key, ChangedBy: changedBy}
}

// EvtAuthEvent is published on token creation, deletion, or validation failure.
type EvtAuthEvent struct {
	base
	AuthKind string `json:"auth_kind"` // "token_created", "token_deleted", "auth_failure"
	Actor    string `json:"actor"`     // label or description of who/what triggered it
}

func (e EvtAuthEvent) Kind() string { return "auth_event" }

// NewAuthEvent constructs an EvtAuthEvent with At = now.
func NewAuthEvent(authKind, actor string) EvtAuthEvent {
	return EvtAuthEvent{base: base{At: now()}, AuthKind: authKind, Actor: actor}
}

// NewAlertFired constructs an EvtAlertFired with At = now.
func NewAlertFired(ruleID int64, ruleName, probeName string, value, threshold float64, severity, condition string, eventID int64) EvtAlertFired {
	return EvtAlertFired{
		base:      base{At: now()},
		RuleID:    ruleID,
		RuleName:  ruleName,
		ProbeName: probeName,
		Value:     value,
		Threshold: threshold,
		Severity:  severity,
		Condition: condition,
		EventID:   eventID,
	}
}

// NewAlertCleared constructs an EvtAlertCleared with At = now.
func NewAlertCleared(ruleID int64, ruleName, probeName string, eventID int64) EvtAlertCleared {
	return EvtAlertCleared{
		base:      base{At: now()},
		RuleID:    ruleID,
		RuleName:  ruleName,
		ProbeName: probeName,
		EventID:   eventID,
	}
}
