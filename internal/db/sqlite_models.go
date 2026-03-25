package db

import "time"

// AuthToken represents a row in the auth_tokens table.
type AuthToken struct {
	ID        int64      `json:"id"`
	Token     string     `json:"-"` // Never expose in list responses
	Label     string     `json:"label"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used"`
}

// ProbeConfig represents a row in the probe_config table.
type ProbeConfig struct {
	ProbeName    string   `json:"probe_name"`
	DisplayName  *string  `json:"display_name"`
	UnitOverride *string  `json:"unit_override"`
	MinNormal    *float64 `json:"min_normal"`
	MaxNormal    *float64 `json:"max_normal"`
	MinWarning   *float64 `json:"min_warning"`
	MaxWarning   *float64 `json:"max_warning"`
	DeviceID     *int64   `json:"device_id"`
}

// Device represents a row in the devices table.
type Device struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DeviceType  *string   `json:"device_type"`
	Description *string   `json:"description"`
	Brand       *string   `json:"brand"`
	Model       *string   `json:"model"`
	Notes       *string   `json:"notes"`
	ImagePath   *string   `json:"image_path"`
	OutletID    *string   `json:"outlet_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ProbeNames  []string  `json:"probe_names"`
}

// OutletConfig represents a row in the outlet_config table.
type OutletConfig struct {
	OutletID    string  `json:"outlet_id"`
	DisplayName *string `json:"display_name"`
	Icon        *string `json:"icon"`
}

// DashboardItem represents a row in the dashboard_items table.
type DashboardItem struct {
	ID          int64   `json:"id"`
	ItemType    string  `json:"item_type"`
	ReferenceID *string `json:"reference_id"`
	Label       *string `json:"label"`
	SortOrder   int     `json:"sort_order"`
}

// AlertRule represents a row in the alert_rules table.
type AlertRule struct {
	ID              int64    `json:"id"`
	ProbeName       string   `json:"probe_name"`
	Condition       string   `json:"condition"`
	ThresholdLow    *float64 `json:"threshold_low"`
	ThresholdHigh   *float64 `json:"threshold_high"`
	Severity        string   `json:"severity"`
	CooldownMinutes int      `json:"cooldown_minutes"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

// OutletEvent represents a row in the outlet_event_log table.
type OutletEvent struct {
	ID          int64     `json:"id"`
	TS          time.Time `json:"ts"`
	OutletID    string    `json:"outlet_id"`
	OutletName  *string   `json:"outlet_name"`
	FromState   *string   `json:"from_state"`
	ToState     string    `json:"to_state"`
	InitiatedBy string   `json:"initiated_by"`
}

// AlertEvent represents a row in the alert_events table.
type AlertEvent struct {
	ID        int64      `json:"id"`
	RuleID    int64      `json:"rule_id"`
	FiredAt   time.Time  `json:"fired_at"`
	ClearedAt *time.Time `json:"cleared_at"`
	PeakValue float64    `json:"peak_value"`
	Notified  bool       `json:"notified"`
	// Joined fields from alert_rules (populated by queries that join).
	ProbeName *string `json:"probe_name,omitempty"`
	Severity  *string `json:"severity,omitempty"`
}

// NotificationTarget represents a row in the notification_targets table.
type NotificationTarget struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Config  string `json:"config"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

// BackupJob represents a row in the backup_jobs table.
type BackupJob struct {
	ID        int64     `json:"id"`
	TS        time.Time `json:"ts"`
	Status    string    `json:"status"`
	Path      *string   `json:"path"`
	SizeBytes *int64    `json:"size_bytes"`
	Error     *string   `json:"error"`
}
