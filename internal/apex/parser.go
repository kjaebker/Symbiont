package apex

import (
	"regexp"
	"strings"
	"time"
)

var switchInputPattern = regexp.MustCompile(`^Sw\d+$`)

// Canonical probe type constants returned by NormalizeProbeType.
const (
	ProbeTypeTemp    = "temp"
	ProbeTypePH      = "ph"
	ProbeTypeAmps    = "amps"
	ProbeTypePower   = "power"
	ProbeTypeVolts   = "volts"
	ProbeTypeDigital = "digital"
	ProbeTypeUnknown = "unknown"
)

// NormalizeProbeType maps an Apex input's type string to a canonical lowercase
// probe type. The Apex uses inconsistent casing and abbreviations:
// "Temp", "pH", "Amps", "pwr", "volts", "digital".
func NormalizeProbeType(input Input) string {
	switch strings.ToLower(input.Type) {
	case "temp":
		return ProbeTypeTemp
	case "ph":
		return ProbeTypePH
	case "amps":
		return ProbeTypeAmps
	case "pwr":
		return ProbeTypePower
	case "volts":
		return ProbeTypeVolts
	case "digital":
		return ProbeTypeDigital
	default:
		return ProbeTypeUnknown
	}
}

// InputClass holds the semantic classification derived from an Apex input at
// ingest time. It is used to populate probe_config defaults so category logic
// stays at the Apex boundary and out of the API layer.
type InputClass struct {
	Category string   // probe | switch | fluid | alarm | virtual
	OnLabel  *string  // label when value != 0 (nil = use default "On")
	OffLabel *string  // label when value == 0 (nil = use default "Off")
	OkValue  *float64 // the value that means "normal"; nil = no auto-status
	IsBinary bool     // true when the input has no meaningful numeric value
	Hidden   bool     // true for inputs with no user-facing meaning by default
}

func strPtr(s string) *string  { return &s }
func f64Ptr(f float64) *float64 { return &f }

// knownInputs maps exact Apex input names to their classification defaults.
var knownInputs = map[string]InputClass{
	"ATOWTR": {Category: "fluid", OnLabel: strPtr("Level OK"), OffLabel: strPtr("Needs Water"), OkValue: f64Ptr(1), IsBinary: true},
	"LEAK":   {Category: "alarm", OnLabel: strPtr("Leak Detected"), OffLabel: strPtr("No Leak"), OkValue: f64Ptr(0), IsBinary: true},
	"FFMNA":  {Category: "fluid", OnLabel: strPtr("Flow Fault"), OffLabel: strPtr("Flow OK"), OkValue: f64Ptr(0), IsBinary: true},
	// Apex virtual outputs that appear as digital inputs.
	"Maintenance":  {Category: "virtual", OnLabel: strPtr("Active"), OffLabel: strPtr("Inactive"), IsBinary: true},
	"MaintWithRet": {Category: "virtual", OnLabel: strPtr("Active"), OffLabel: strPtr("Inactive"), IsBinary: true},
	"PhotoMode":    {Category: "virtual", OnLabel: strPtr("Active"), OffLabel: strPtr("Inactive"), IsBinary: true},
}

// ClassifyInput derives the semantic InputClass for an Apex Input. The result
// represents what is known at ingest time and should be used as the default
// when first creating a probe_config row. User overrides are preserved on
// subsequent upserts (callers should only apply this on first insert).
func ClassifyInput(input Input) InputClass {
	if known, ok := knownInputs[input.Name]; ok {
		return known
	}
	if NormalizeProbeType(input) == ProbeTypeDigital {
		if switchInputPattern.MatchString(input.Name) {
			return InputClass{Category: "switch", IsBinary: true, Hidden: true}
		}
		// Unrecognised digital input — treat as a generic switch.
		return InputClass{Category: "switch", IsBinary: true}
	}
	return InputClass{Category: "probe"}
}

// PowerEvent represents a parsed power failure or restoration event.
type PowerEvent struct {
	Type      string    // "failed" or "restored"
	Timestamp time.Time // UTC time of the event
}

// ParsePowerEvents converts the PowerInfo Unix epoch timestamps into a slice
// of PowerEvent values. Zero timestamps are skipped (no event recorded).
func ParsePowerEvents(power PowerInfo) []PowerEvent {
	var events []PowerEvent

	if power.Failed != 0 {
		events = append(events, PowerEvent{
			Type:      "failed",
			Timestamp: time.Unix(power.Failed, 0).UTC(),
		})
	}

	if power.Restored != 0 {
		events = append(events, PowerEvent{
			Type:      "restored",
			Timestamp: time.Unix(power.Restored, 0).UTC(),
		})
	}

	return events
}

// OutletPower holds the power draw readings for a single outlet.
type OutletPower struct {
	Amps  float64 // current draw in amps
	Watts float64 // power draw in watts
}

// CorrelateOutletPower matches per-outlet amp and watt input entries to outlets
// by the Apex naming convention: <OutletName>A for amps and <OutletName>W for
// watts. Returns a map keyed by outlet DID.
func CorrelateOutletPower(inputs []Input, outputs []Output) map[string]OutletPower {
	// Build a map of outlet name -> DID for lookups.
	nameToOutput := make(map[string]string, len(outputs))
	for _, o := range outputs {
		nameToOutput[o.Name] = o.DID
	}

	// Build per-DID power readings by matching input names.
	result := make(map[string]OutletPower)

	for _, input := range inputs {
		name := input.Name

		// Check for amps suffix: <OutletName>A
		if strings.HasSuffix(name, "A") {
			outletName := strings.TrimSuffix(name, "A")
			if did, ok := nameToOutput[outletName]; ok {
				p := result[did]
				p.Amps = input.Value
				result[did] = p
			}
		}

		// Check for watts suffix: <OutletName>W
		if strings.HasSuffix(name, "W") {
			outletName := strings.TrimSuffix(name, "W")
			if did, ok := nameToOutput[outletName]; ok {
				p := result[did]
				p.Watts = input.Value
				result[did] = p
			}
		}
	}

	return result
}
