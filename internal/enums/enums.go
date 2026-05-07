// Package enums centralises every constrained-string vocabulary used by the
// API. Each Set lists the values the schema's CHECK constraint accepts, and
// handlers consult these sets rather than maintaining their own
// map[string]bool copies.
//
// When adding a value to a CHECK constraint you must also extend the matching
// Set here (and the matching TypeScript type in the frontend). The
// schema_drift test in this package fails the build if a Set and its
// migration disagree.
package enums

import "sort"

// Set is an immutable membership-tested collection of allowed string values.
// The zero value is unusable; construct sets via NewSet.
type Set struct {
	m      map[string]struct{}
	values []string
}

// NewSet returns a Set containing the provided values. Duplicates are
// silently collapsed.
func NewSet(values ...string) Set {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return Set{m: m, values: out}
}

// Has reports whether v is in the set.
func (s Set) Has(v string) bool {
	_, ok := s.m[v]
	return ok
}

// Values returns the set's values in deterministic (sorted) order.
func (s Set) Values() []string {
	out := make([]string, len(s.values))
	copy(out, s.values)
	return out
}

// Livestock vocabularies — see CHECK on `livestock` and `livestock_observations`.
var (
	LivestockTypes    = NewSet("fish", "coral", "invertebrate", "other")
	LivestockStatuses = NewSet("healthy", "sick", "quarantine", "deceased")
)

// Journal vocabularies — see CHECK on `journal_entries`.
var (
	JournalCategories = NewSet("observation", "maintenance", "event", "milestone")
	JournalSentiments = NewSet("good", "neutral", "bad", "critical")
	JournalSources    = NewSet("manual", "system", "ai")
)

// Agent vocabularies — see CHECK on `agent_settings.tone`. DosingProductLines
// is enforced by the API handler only; the column is untyped.
var (
	AgentTones         = NewSet("analytical", "casual", "terse")
	DosingProductLines = NewSet("brs_pharma", "red_sea", "tropic_marin", "generic", "none")
)

// TokenScopes — see CHECK on `auth_tokens.scope`.
var TokenScopes = NewSet("read", "control", "admin")

// TankTypes — see CHECK on `tank_profile.tank_type`.
var TankTypes = NewSet("reef", "fowlr", "mixed", "nano", "freshwater", "other")

// DeviceTypes — currently API-side only; the `devices.device_type` column is
// untyped because user-supplied types are tolerated.
var DeviceTypes = NewSet(
	"heater", "pump", "wavemaker", "light", "skimmer",
	"reactor", "doser", "ato", "chiller", "fan", "other",
)

// DashboardItemTypes — see CHECK on `dashboard_items.item_type`.
var DashboardItemTypes = NewSet("probe", "outlet", "device", "separator", "feed_mode", "measurement")

// HistoryIntervals — bucket sizes accepted by /api/probes/{name}/history and
// /api/outlets/{id}/history. No DB equivalent.
var HistoryIntervals = NewSet("10s", "30s", "1m", "5m", "15m", "1h", "1d")

// ImageExtensions — accepted file extensions for image uploads (lowercase).
var ImageExtensions = NewSet(".jpg", ".jpeg", ".png", ".webp")

// InitiatedBy enumerates which interface caused a state change. Stored in
// audit-event payloads under `initiated_by`. Distinct from a row's `source`
// column (which records provenance like manual/system/ai).
var InitiatedBy = NewSet("ui", "cli", "mcp", "api")
