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
	DisplayMode string  `json:"display_mode"` // "normal" or "compact"
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

// MeasurementParameter represents a row in the measurement_parameters table.
type MeasurementParameter struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CanonicalUnit string `json:"canonical_unit"`
	SortOrder     int    `json:"sort_order"`
}

// MeasurementFilter is used to filter the ListMeasurements query.
type MeasurementFilter struct {
	ParameterID *int64
	From        *time.Time
	To          *time.Time
	Limit       int // defaults to 200 if <= 0
}

// Measurement represents a row in the measurements table, joined with its parameter name and unit.
type Measurement struct {
	ID          int64     `json:"id"`
	MeasuredAt  time.Time `json:"measured_at"`
	ParameterID int64     `json:"parameter_id"`
	Value       float64   `json:"value"`
	Notes       *string   `json:"notes"`
	Source      string    `json:"source"`
	TestKitRef  *string   `json:"test_kit_ref"`
	RawValue    *float64  `json:"raw_value"`
	CreatedAt   time.Time `json:"created_at"`
	// Joined from measurement_parameters:
	Parameter     string `json:"parameter"`
	CanonicalUnit string `json:"canonical_unit"`
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

// LivestockItem represents a row in the livestock table.
type LivestockItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Species   *string   `json:"species"`
	Type      string    `json:"type"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	DateAdded *string   `json:"date_added"` // YYYY-MM-DD text
	Notes     *string   `json:"notes"`
	ImagePath *string   `json:"image_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LivestockObservation represents a row in the livestock_observations table.
type LivestockObservation struct {
	ID          int64     `json:"id"`
	LivestockID int64     `json:"livestock_id"`
	TS          time.Time `json:"ts"`
	Status      *string   `json:"status"`
	Note        *string   `json:"note"`
	ImagePath   *string   `json:"image_path"`
}

// LivestockFilter is used to filter ListLivestock queries.
type LivestockFilter struct {
	Type   string // empty = all
	Status string // empty = all
}

// TankProfile represents a row in the tank_profile table.
// Section is "display" or "sump" and serves as the primary key.
type TankProfile struct {
	Section       string    `json:"section"`
	DisplayName   *string   `json:"display_name"`
	VolumeGallons *float64  `json:"volume_gallons"`
	LengthIn      *float64  `json:"length_in"`
	WidthIn       *float64  `json:"width_in"`
	HeightIn      *float64  `json:"height_in"`
	TankType      *string   `json:"tank_type"`
	Manufacturer  *string   `json:"manufacturer"`
	Model         *string   `json:"model"`
	SetupDate     *string   `json:"setup_date"` // YYYY-MM-DD
	Notes         *string   `json:"notes"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// JournalEntry represents a row in the journal_entries table.
type JournalEntry struct {
	ID        int64     `json:"id"`
	TS        time.Time `json:"ts"`
	Category  string    `json:"category"`  // observation | maintenance | event | milestone
	Sentiment *string   `json:"sentiment"` // good | neutral | bad | critical
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
	Source    string    `json:"source"`     // manual | system | ai
	SourceRef *string   `json:"source_ref"` // e.g. "feed_mode:1" for system entries
	CreatedAt time.Time `json:"created_at"`
}

// JournalFilter is used to filter ListJournalEntries queries.
type JournalFilter struct {
	Category  string
	Sentiment string
	From      *time.Time
	To        *time.Time
	Limit     int // defaults to 50 if <= 0
}

// AgentSettings represents the single-row agent_settings table. It stores the
// tunable persona parameters used by get_agent_context to assemble a system
// prompt for external Claude clients.
type AgentSettings struct {
	Tone              string   `json:"tone"`                // analytical | casual | terse
	DosingProductLine string   `json:"dosing_product_line"` // brs_pharma | red_sea | tropic_marin | generic | none
	NetVolumeGallons  *float64 `json:"net_volume_gallons"`  // override; nil = derive from tank_profile
	CustomGuardrails  *string  `json:"custom_guardrails"`
	EnabledSkills     []string `json:"enabled_skills"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AuditEvent represents a row in the events table — a forensic record of every
// significant state change that flows through the system event bus.
type AuditEvent struct {
	ID            int64     `json:"id"`
	TS            time.Time `json:"ts"`
	Kind          string    `json:"kind"`
	PayloadJSON   string    `json:"payload_json"`
	CorrelationID *string   `json:"correlation_id,omitempty"`
}

// AuditFilter controls which rows ListAuditEvents returns.
type AuditFilter struct {
	Kind          string     // empty = all kinds
	Since         *time.Time // inclusive lower bound
	Limit         int        // defaults to 100 if <= 0; max 500
	CorrelationID string     // empty = all; filters on correlation_id column
	InitiatedBy   string     // empty = all; filters on json_extract(payload_json, '$.data.initiated_by')
}
