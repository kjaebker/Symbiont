package apex

import (
	"strings"
	"time"
)

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
