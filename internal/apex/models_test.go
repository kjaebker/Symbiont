package apex

import (
	"encoding/json"
	"os"
	"testing"
)

func TestStatusResponseDeserialization(t *testing.T) {
	data, err := os.ReadFile("../../testdata/status-response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var resp StatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	// System
	if resp.System.Hostname != "apex" {
		t.Errorf("system.hostname = %q, want %q", resp.System.Hostname, "apex")
	}
	if resp.System.Software != "5.12L_CA25" {
		t.Errorf("system.software = %q, want %q", resp.System.Software, "5.12L_CA25")
	}
	if resp.System.Serial != "AC6L:4034" {
		t.Errorf("system.serial = %q, want %q", resp.System.Serial, "AC6L:4034")
	}
	if resp.System.Type != "AC6L" {
		t.Errorf("system.type = %q, want %q", resp.System.Type, "AC6L")
	}
	if resp.System.Date != 1774108274 {
		t.Errorf("system.date = %d, want %d", resp.System.Date, 1774108274)
	}
	if resp.System.Timezone != "-4.00" {
		t.Errorf("system.timezone = %q, want %q", resp.System.Timezone, "-4.00")
	}

	// Power (top-level, not in system)
	if resp.Power.Failed != 1770564515 {
		t.Errorf("power.failed = %d, want %d", resp.Power.Failed, 1770564515)
	}
	if resp.Power.Restored != 1770564550 {
		t.Errorf("power.restored = %d, want %d", resp.Power.Restored, 1770564550)
	}

	// Inputs
	if len(resp.Inputs) != 28 {
		t.Fatalf("len(inputs) = %d, want 28", len(resp.Inputs))
	}
	temp := resp.Inputs[0]
	if temp.DID != "base_Temp" || temp.Type != "Temp" || temp.Name != "Tmp" || temp.Value != 78.4 {
		t.Errorf("inputs[0] = %+v, want base_Temp/Temp/Tmp/78.4", temp)
	}
	ph := resp.Inputs[1]
	if ph.DID != "base_pH" || ph.Type != "pH" || ph.Value != 8.38 {
		t.Errorf("inputs[1] = %+v, want base_pH/pH/8.38", ph)
	}

	// Outputs
	if len(resp.Outputs) != 23 {
		t.Fatalf("len(outputs) = %d, want 23", len(resp.Outputs))
	}

	// Find SumpFlow by DID to avoid brittle index assumptions
	var sumpFlow *Output
	var kessil *Output
	for i := range resp.Outputs {
		switch resp.Outputs[i].DID {
		case "2_1":
			sumpFlow = &resp.Outputs[i]
		case "3_1":
			kessil = &resp.Outputs[i]
		}
	}

	// Test an outlet type
	if sumpFlow == nil {
		t.Fatal("outlet with did=2_1 (SumpFlow) not found")
	}
	if sumpFlow.Name != "SumpFlow" {
		t.Errorf("SumpFlow.name = %q, want %q", sumpFlow.Name, "SumpFlow")
	}
	if sumpFlow.Type != "outlet" {
		t.Errorf("SumpFlow.type = %q, want %q", sumpFlow.Type, "outlet")
	}
	if sumpFlow.State() != "AON" {
		t.Errorf("SumpFlow.State() = %q, want %q", sumpFlow.State(), "AON")
	}
	if sumpFlow.Health() != "OK" {
		t.Errorf("SumpFlow.Health() = %q, want %q", sumpFlow.Health(), "OK")
	}

	// Test a variable type with intensity
	if kessil == nil {
		t.Fatal("output with did=3_1 (KessilColor) not found")
	}
	if kessil.Name != "KessilColor" {
		t.Errorf("KessilColor.name = %q, want %q", kessil.Name, "KessilColor")
	}
	if kessil.Type != "variable" {
		t.Errorf("KessilColor.type = %q, want %q", kessil.Type, "variable")
	}
	if kessil.State() != "TBL" {
		t.Errorf("KessilColor.State() = %q, want %q", kessil.State(), "TBL")
	}
	if kessil.Intensity == nil || *kessil.Intensity != 48 {
		t.Errorf("KessilColor.intensity = %v, want 48", kessil.Intensity)
	}

	// Modules
	if len(resp.Modules) != 3 {
		t.Fatalf("len(modules) = %d, want 3", len(resp.Modules))
	}
	if resp.Modules[1].HWType != "EB832" {
		t.Errorf("modules[1].hwtype = %q, want %q", resp.Modules[1].HWType, "EB832")
	}

	// Feed
	if resp.Feed.Name != 0 || resp.Feed.Active != 0 {
		t.Errorf("feed = %+v, want name=0 active=0", resp.Feed)
	}

	// Link
	if resp.Link.LinkState != 3 {
		t.Errorf("link.linkState = %d, want 3", resp.Link.LinkState)
	}

	// Nstat
	if resp.Nstat.IPAddr != "192.168.1.127" {
		t.Errorf("nstat.ipaddr = %q, want %q", resp.Nstat.IPAddr, "192.168.1.127")
	}
}

func TestOutputStateAndHealth(t *testing.T) {
	tests := []struct {
		name       string
		status     []string
		wantState  string
		wantHealth string
	}{
		{"full status", []string{"AON", "", "OK", ""}, "AON", "OK"},
		{"forced off", []string{"OFF", "", "---", ""}, "OFF", "---"},
		{"empty status", []string{}, "", ""},
		{"short status", []string{"ON"}, "ON", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Output{Status: tt.status}
			if got := o.State(); got != tt.wantState {
				t.Errorf("State() = %q, want %q", got, tt.wantState)
			}
			if got := o.Health(); got != tt.wantHealth {
				t.Errorf("Health() = %q, want %q", got, tt.wantHealth)
			}
		})
	}
}

func TestNewOutletControlRequest(t *testing.T) {
	req := NewOutletControlRequest("2_1", OutletOn)

	if req.DID != "2_1" {
		t.Errorf("DID = %q, want %q", req.DID, "2_1")
	}
	if req.Type != "outlet" {
		t.Errorf("Type = %q, want %q", req.Type, "outlet")
	}
	if len(req.Status) != 4 {
		t.Fatalf("len(Status) = %d, want 4", len(req.Status))
	}
	if req.Status[0] != "ON" {
		t.Errorf("Status[0] = %q, want %q", req.Status[0], "ON")
	}
	if req.Status[2] != "OK" {
		t.Errorf("Status[2] = %q, want %q", req.Status[2], "OK")
	}

	// Verify it serializes correctly
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"did":"2_1","status":["ON","","OK",""],"type":"outlet"}`
	if string(data) != want {
		t.Errorf("JSON = %s, want %s", data, want)
	}
}
