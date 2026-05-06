package db

import "time"

// AuthToken represents a row in the auth_tokens table.
type AuthToken struct {
	ID        int64      `json:"id"`
	Token     string     `json:"-"` // Never expose in list responses
	Label     string     `json:"label"`
	Scope     string     `json:"scope"` // read | control | admin
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used"`
}

// TokenAuth is returned by ValidateToken and holds the data the auth
// middleware needs without exposing the raw token string.
type TokenAuth struct {
	ID    int64
	Label string
	Scope string // read | control | admin
}

// ProbeConfig represents a row in the probe_config table.
type ProbeConfig struct {
	ProbeName     string   `json:"probe_name"`
	DisplayName   *string  `json:"display_name"`
	UnitOverride  *string  `json:"unit_override"`
	MinNormal     *float64 `json:"min_normal"`
	MaxNormal     *float64 `json:"max_normal"`
	MinWarning    *float64 `json:"min_warning"`
	MaxWarning    *float64 `json:"max_warning"`
	DeviceID      *int64   `json:"device_id"`
	InputCategory string   `json:"input_category"` // probe | switch | fluid | alarm | virtual
	OnLabel       *string  `json:"on_label"`
	OffLabel      *string  `json:"off_label"`
	OkValue       *float64 `json:"ok_value"` // value meaning "normal" for binary inputs
	IsBinary      bool     `json:"is_binary"`
	Hidden        bool     `json:"hidden"`
}

// DeviceOutlet represents a row in the device_outlets table.
// These are the visualization outlets linked to a device (e.g. Kessil Color + Power
// channels), separate from Device.OutletID which is used for ON/OFF/AUTO control.
type DeviceOutlet struct {
	DeviceID  int64   `json:"device_id"`
	OutletID  string  `json:"outlet_id"`
	Label     *string `json:"label"`
	Color     *string `json:"color"`
	SortOrder int     `json:"sort_order"`
}

// Device represents a row in the devices table.
type Device struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	DeviceType  *string        `json:"device_type"`
	Description *string        `json:"description"`
	Brand       *string        `json:"brand"`
	Model       *string        `json:"model"`
	Notes       *string        `json:"notes"`
	ImagePath   *string        `json:"image_path"`
	OutletID    *string        `json:"outlet_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ProbeNames  []string       `json:"probe_names"`
	OutletIDs   []DeviceOutlet `json:"outlet_ids"`
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

// ImageRecord is used by the reprocess migration to enumerate all stored images
// with their owning IDs so deterministic filenames can be computed.
type ImageRecord struct {
	Kind        string // "livestock" or "observation"
	ID          int64  // livestock item ID or observation ID
	LivestockID int64  // same as ID for livestock; parent livestock ID for observations
	ImagePath   string
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
	InitiatedBy   *string   `json:"initiated_by,omitempty"`
}

// DosingProduct represents a row in the dosing_products table.
type DosingProduct struct {
	ID        int64     `json:"id"`
	Brand     string    `json:"brand"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Unit      string    `json:"unit"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// DosingSchedule represents a row in the dosing_schedules table, joined with product info.
type DosingSchedule struct {
	ID                  int64      `json:"id"`
	ProductID           int64      `json:"product_id"`
	Amount              float64    `json:"amount"`
	Frequency           string     `json:"frequency"`
	IntervalDays        *float64   `json:"interval_days"`
	DayOfWeek           *int       `json:"day_of_week"`
	Enabled             bool       `json:"enabled"`
	LastCompletedAt     *time.Time `json:"last_completed_at"`
	NextDueAt           *time.Time `json:"next_due_at"`
	FollowUpParameterID *int64     `json:"follow_up_parameter_id"`
	FollowUpDays        *int       `json:"follow_up_days"`
	Notes               *string    `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	// Joined from dosing_products:
	ProductBrand *string `json:"product_brand,omitempty"`
	ProductName  *string `json:"product_name,omitempty"`
	ProductUnit  *string `json:"product_unit,omitempty"`
	// Joined from measurement_parameters:
	FollowUpParameter *string `json:"follow_up_parameter,omitempty"`
}

// DosingLog represents a row in the dosing_logs table, joined with product info.
type DosingLog struct {
	ID         int64     `json:"id"`
	ScheduleID *int64    `json:"schedule_id"`
	ProductID  int64     `json:"product_id"`
	Amount     float64   `json:"amount"`
	DosedAt    time.Time `json:"dosed_at"`
	Notes      *string   `json:"notes"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
	// Joined from dosing_products:
	ProductBrand string `json:"product_brand"`
	ProductName  string `json:"product_name"`
	ProductUnit  string `json:"product_unit"`
}

// DosingLogFilter filters ListDosingLogs queries.
type DosingLogFilter struct {
	ProductID  *int64
	ScheduleID *int64
	From       *time.Time
	To         *time.Time
	Limit      int // defaults to 100 if <= 0
}

// MaintenanceTask represents a row in the maintenance_tasks table.
type MaintenanceTask struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	Frequency       string     `json:"frequency"`
	IntervalDays    *float64   `json:"interval_days"`
	DayOfWeek       *int       `json:"day_of_week"`
	Enabled         bool       `json:"enabled"`
	LastCompletedAt *time.Time `json:"last_completed_at"`
	NextDueAt       *time.Time `json:"next_due_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// MaintenanceLog represents a row in the maintenance_logs table.
type MaintenanceLog struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	CompletedAt time.Time `json:"completed_at"`
	Notes       *string   `json:"notes"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
	// Joined from maintenance_tasks:
	TaskName *string `json:"task_name,omitempty"`
}

// DueItem represents a single due or overdue dosing/maintenance item.
type DueItem struct {
	Kind        string     `json:"kind"` // "dose" | "task" | "followup"
	ID          int64      `json:"id"`   // schedule_id or task_id
	Label       string     `json:"label"`
	Detail      string     `json:"detail"` // e.g. "5 mL" or "every 7 days"
	NextDueAt   *time.Time `json:"next_due_at"`
	IsOverdue   bool       `json:"is_overdue"`
	ProductID   *int64     `json:"product_id,omitempty"`   // for dose items
	ProductUnit *string    `json:"product_unit,omitempty"` // for dose items
	Amount      *float64   `json:"amount,omitempty"`       // for dose items
}

// AuditFilter controls which rows ListAuditEvents returns.
type AuditFilter struct {
	Kind          string     // empty = all kinds
	Since         *time.Time // inclusive lower bound
	Limit         int        // defaults to 100 if <= 0; max 500
	CorrelationID string     // empty = all; filters on correlation_id column
	InitiatedBy   string     // empty = all; filters on json_extract(payload_json, '$.data.initiated_by')
}
