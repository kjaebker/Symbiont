package apex

import (
	"testing"
	"time"
)

func TestNormalizeProbeType(t *testing.T) {
	tests := []struct {
		name  string
		input Input
		want  string
	}{
		{"Temp capitalized", Input{Type: "Temp"}, ProbeTypeTemp},
		{"pH mixed case", Input{Type: "pH"}, ProbeTypePH},
		{"Amps capitalized", Input{Type: "Amps"}, ProbeTypeAmps},
		{"pwr lowercase", Input{Type: "pwr"}, ProbeTypePower},
		{"volts lowercase", Input{Type: "volts"}, ProbeTypeVolts},
		{"digital lowercase", Input{Type: "digital"}, ProbeTypeDigital},
		{"TEMP all caps", Input{Type: "TEMP"}, ProbeTypeTemp},
		{"unknown type", Input{Type: "ORP"}, ProbeTypeUnknown},
		{"empty type", Input{Type: ""}, ProbeTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeProbeType(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeProbeType(%q) = %q, want %q", tt.input.Type, got, tt.want)
			}
		})
	}
}

func TestParsePowerEvents(t *testing.T) {
	t.Run("both events present", func(t *testing.T) {
		power := PowerInfo{
			Failed:   1770564515,
			Restored: 1770564550,
		}
		events := ParsePowerEvents(power)
		if len(events) != 2 {
			t.Fatalf("len(events) = %d, want 2", len(events))
		}

		if events[0].Type != "failed" {
			t.Errorf("events[0].Type = %q, want %q", events[0].Type, "failed")
		}
		wantFailed := time.Unix(1770564515, 0).UTC()
		if !events[0].Timestamp.Equal(wantFailed) {
			t.Errorf("events[0].Timestamp = %v, want %v", events[0].Timestamp, wantFailed)
		}

		if events[1].Type != "restored" {
			t.Errorf("events[1].Type = %q, want %q", events[1].Type, "restored")
		}
		wantRestored := time.Unix(1770564550, 0).UTC()
		if !events[1].Timestamp.Equal(wantRestored) {
			t.Errorf("events[1].Timestamp = %v, want %v", events[1].Timestamp, wantRestored)
		}
	})

	t.Run("no events", func(t *testing.T) {
		power := PowerInfo{Failed: 0, Restored: 0}
		events := ParsePowerEvents(power)
		if len(events) != 0 {
			t.Errorf("len(events) = %d, want 0", len(events))
		}
	})

	t.Run("only failed", func(t *testing.T) {
		power := PowerInfo{Failed: 1770564515, Restored: 0}
		events := ParsePowerEvents(power)
		if len(events) != 1 {
			t.Fatalf("len(events) = %d, want 1", len(events))
		}
		if events[0].Type != "failed" {
			t.Errorf("events[0].Type = %q, want %q", events[0].Type, "failed")
		}
	})
}

func TestCorrelateOutletPower(t *testing.T) {
	inputs := []Input{
		{DID: "base_Temp", Type: "Temp", Name: "Tmp", Value: 78.4},
		{DID: "base_pH", Type: "pH", Name: "pH", Value: 8.38},
		{DID: "2_0_Amps", Type: "Amps", Name: "ReturnPmpA", Value: 0.4},
		{DID: "2_0_Watts", Type: "pwr", Name: "ReturnPmpW", Value: 45.2},
		{DID: "2_1_Amps", Type: "Amps", Name: "SumpFlowA", Value: 0.3},
		{DID: "2_1_Watts", Type: "pwr", Name: "SumpFlowW", Value: 32.1},
		{DID: "2_2_Amps", Type: "Amps", Name: "HeaterA", Value: 0.0},
		{DID: "2_2_Watts", Type: "pwr", Name: "HeaterW", Value: 0.0},
	}

	outputs := []Output{
		{DID: "2_0", Name: "ReturnPmp", Type: "outlet"},
		{DID: "2_1", Name: "SumpFlow", Type: "outlet"},
		{DID: "2_2", Name: "Heater", Type: "outlet"},
		{DID: "2_3", Name: "Skimmer", Type: "outlet"},
	}

	result := CorrelateOutletPower(inputs, outputs)

	// ReturnPmp should have both amps and watts.
	if p, ok := result["2_0"]; !ok {
		t.Error("ReturnPmp (2_0) not found in result")
	} else {
		if p.Amps != 0.4 {
			t.Errorf("ReturnPmp amps = %f, want 0.4", p.Amps)
		}
		if p.Watts != 45.2 {
			t.Errorf("ReturnPmp watts = %f, want 45.2", p.Watts)
		}
	}

	// SumpFlow should have both.
	if p, ok := result["2_1"]; !ok {
		t.Error("SumpFlow (2_1) not found in result")
	} else {
		if p.Amps != 0.3 {
			t.Errorf("SumpFlow amps = %f, want 0.3", p.Amps)
		}
		if p.Watts != 32.1 {
			t.Errorf("SumpFlow watts = %f, want 32.1", p.Watts)
		}
	}

	// Heater should have zero values.
	if p, ok := result["2_2"]; !ok {
		t.Error("Heater (2_2) not found in result")
	} else {
		if p.Amps != 0.0 {
			t.Errorf("Heater amps = %f, want 0.0", p.Amps)
		}
	}

	// Skimmer has no matching inputs — should not be in result.
	if _, ok := result["2_3"]; ok {
		t.Error("Skimmer (2_3) should not be in result without matching inputs")
	}

	// Total count: 3 outlets with power data.
	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
}

func TestCorrelateOutletPowerEmpty(t *testing.T) {
	result := CorrelateOutletPower(nil, nil)
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}
