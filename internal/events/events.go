// Package events defines the typed system events that flow through the event bus.
// SystemEvent is a sealed interface — only types defined in this package can
// implement it, ensuring all event kinds are enumerable and auditable.
package events

import "time"

// SystemEvent is the sealed interface for all system events.
// The unexported sealed() method prevents implementation outside this package.
type SystemEvent interface {
	Kind() string
	OccurredAt() time.Time
	sealed()
}

// base embeds common fields and satisfies the sealed() constraint.
type base struct {
	At time.Time
}

func (b base) OccurredAt() time.Time { return b.At }
func (b base) sealed()               {}

// now returns the current time, used as a default for events that don't supply At.
func now() time.Time { return time.Now() }

// EvtFeedModeActivated is published when a Neptune Apex feed mode is activated.
type EvtFeedModeActivated struct {
	base
	Mode    string // "A", "B", "C", or "D"
	FeedNum int    // 1–4
}

func (e EvtFeedModeActivated) Kind() string { return "feed_mode_activated" }

// NewFeedModeActivated constructs an EvtFeedModeActivated with At = now.
func NewFeedModeActivated(mode string, feedNum int) EvtFeedModeActivated {
	return EvtFeedModeActivated{base: base{At: now()}, Mode: mode, FeedNum: feedNum}
}

// EvtOutletChanged is published after a successful outlet state change.
type EvtOutletChanged struct {
	base
	OutletID    string
	Name        string
	PrevState   string // empty string if unknown
	NewState    string
	InitiatedBy string // "ui", "cli", "mcp", "api", "apex"
}

func (e EvtOutletChanged) Kind() string { return "outlet_changed" }

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
	RuleID      int64
	RuleName    string // probe display name or probe name
	ProbeName   string
	Value       float64
	Threshold   float64 // the relevant threshold that was exceeded
	Severity    string
	Condition   string
	EventID     int64 // alert_events.id
}

func (e EvtAlertFired) Kind() string { return "alert_fired" }

// EvtAlertCleared is published when a previously fired alert resolves.
type EvtAlertCleared struct {
	base
	RuleID    int64
	RuleName  string
	ProbeName string
	EventID   int64
}

func (e EvtAlertCleared) Kind() string { return "alert_cleared" }

// EvtObservationRecorded is published when a livestock observation is created.
type EvtObservationRecorded struct {
	base
	ObservationID int64
	LivestockID   int64
	LivestockName string
	Status        string // may be empty if observation has no status
	PrevStatus    string // previous livestock status
	HasImage      bool
}

func (e EvtObservationRecorded) Kind() string { return "observation_recorded" }

// EvtLivestockAdded is published after a new livestock item is inserted.
type EvtLivestockAdded struct {
	base
	LivestockID int64
	Name        string
	Species     string // may be empty
	Type        string // fish, coral, invertebrate, other
}

func (e EvtLivestockAdded) Kind() string { return "livestock_added" }

// EvtLivestockUpdated is published after a livestock item's fields change.
type EvtLivestockUpdated struct {
	base
	LivestockID   int64
	Name          string
	ChangedFields []string // e.g. ["status", "quantity"]
}

func (e EvtLivestockUpdated) Kind() string { return "livestock_updated" }

// EvtJournalEntryCreated is published after a journal entry is inserted.
type EvtJournalEntryCreated struct {
	base
	EntryID  int64
	Category string
	Source   string // "manual", "system", "ai"
}

func (e EvtJournalEntryCreated) Kind() string { return "journal_entry_created" }

// EvtPollCycleCompleted is published after each successful Apex poll cycle.
type EvtPollCycleCompleted struct {
	base
	DurationMs int64
	ProbeCount int
}

func (e EvtPollCycleCompleted) Kind() string { return "poll_cycle_completed" }

// EvtConfigChanged is published when a probe or outlet config is updated.
type EvtConfigChanged struct {
	base
	Key       string // e.g. "probe:Tmp" or "outlet:3_1"
	ChangedBy string // initiator description
}

func (e EvtConfigChanged) Kind() string { return "config_changed" }

// EvtAuthEvent is published on token creation, deletion, or validation failure.
type EvtAuthEvent struct {
	base
	AuthKind string // "token_created", "token_deleted", "auth_failure"
	Actor    string // label or description of who/what triggered it
}

func (e EvtAuthEvent) Kind() string { return "auth_event" }
