package db

import "testing"

func TestProbeStatus(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	analog := &ProbeConfig{MinNormal: f(76), MaxNormal: f(80), MinWarning: f(74), MaxWarning: f(82)}
	leak := &ProbeConfig{IsBinary: true, OkValue: f(0), InputCategory: "alarm"}
	float := &ProbeConfig{IsBinary: true, OkValue: f(1), InputCategory: "fluid"}

	tests := []struct {
		name  string
		cfg   *ProbeConfig
		value float64
		want  string
	}{
		{"no config", nil, 78, ProbeStatusUnknown},
		{"no ranges", &ProbeConfig{}, 78, ProbeStatusUnknown},
		{"in range", analog, 78, ProbeStatusNormal},
		{"above normal", analog, 81, ProbeStatusWarning},
		{"below normal", analog, 75, ProbeStatusWarning},
		{"above critical limit", analog, 83, ProbeStatusCritical},
		{"below critical limit", analog, 73, ProbeStatusCritical},
		{"alarm ok", leak, 0, ProbeStatusNormal},
		{"alarm tripped", leak, 1, ProbeStatusCritical},
		{"fluid not ok", float, 0, ProbeStatusWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProbeStatus(tt.value, tt.cfg); got != tt.want {
				t.Errorf("ProbeStatus() = %q, want %q", got, tt.want)
			}
		})
	}

	var none *ProbeConfig
	if none.CanReach("warning") || (&ProbeConfig{MinNormal: f(1)}).CanReach("critical") || float.CanReach("critical") {
		t.Error("CanReach reported a severity the config cannot produce")
	}
	if !analog.CanReach("critical") || !leak.CanReach("critical") || !float.CanReach("warning") {
		t.Error("CanReach missed a severity the config can produce")
	}
}
