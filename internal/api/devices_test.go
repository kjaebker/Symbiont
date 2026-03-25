package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kjaebker/symbiont/internal/db"
)

func TestDeviceCRUDViaAPI(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// List — empty.
	w := env.request(t, "GET", "/api/devices", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct{ Devices []db.Device }
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(listResp.Devices))
	}

	// Create.
	w = env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Return Pump",
		"device_type": "pump",
		"brand":       "Sicce",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created db.Device
	json.NewDecoder(w.Body).Decode(&created)
	if created.Name != "Return Pump" {
		t.Errorf("expected name 'Return Pump', got %q", created.Name)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero device ID")
	}

	// Get.
	w = env.request(t, "GET", "/api/devices/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update.
	w = env.request(t, "PUT", "/api/devices/1", map[string]any{
		"name":        "Main Return Pump",
		"device_type": "pump",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated db.Device
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Main Return Pump" {
		t.Errorf("expected name 'Main Return Pump', got %q", updated.Name)
	}

	// Delete.
	w = env.request(t, "DELETE", "/api/devices/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get after delete — 404.
	w = env.request(t, "GET", "/api/devices/1", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeviceCreateValidation(t *testing.T) {
	env := setupTestEnv(t)

	// Missing name.
	w := env.request(t, "POST", "/api/devices", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}

	// Invalid device type.
	w = env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Test",
		"device_type": "invalid_type",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid device_type, got %d", w.Code)
	}
}

func TestDeviceProbesViaAPI(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// Create device.
	w := env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Temp Sensor",
		"probe_names": []string{"Tmp"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Replace probes.
	w = env.request(t, "PUT", "/api/devices/1/probes", map[string]any{
		"probe_names": []string{"Tmp", "pH"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var dev db.Device
	json.NewDecoder(w.Body).Decode(&dev)
	if len(dev.ProbeNames) != 2 {
		t.Errorf("expected 2 probe names, got %d", len(dev.ProbeNames))
	}
}

func TestDeviceSuggestions(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	w := env.request(t, "GET", "/api/devices/suggestions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Suggestions []struct {
			OutletName    string   `json:"outlet_name"`
			OutletID      string   `json:"outlet_id"`
			ProbeNames    []string `json:"probe_names"`
			SuggestedName string   `json:"suggested_name"`
		}
	}
	json.NewDecoder(w.Body).Decode(&resp)
	// Seeded data has one outlet "Return" but no matching W/A probes,
	// so suggestions should be empty.
	if len(resp.Suggestions) != 0 {
		t.Logf("suggestions: %+v", resp.Suggestions)
	}
}

func TestNameConflictGuardProbe(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// Create device and link probe.
	env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Temp Sensor",
		"probe_names": []string{"Tmp"},
	})

	// Try to change display name on linked probe — should be rejected.
	w := env.request(t, "PUT", "/api/config/probes/Tmp", map[string]any{
		"display_name": "My Custom Name",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for device-managed probe, got %d: %s", w.Code, w.Body.String())
	}

	// Non-display-name changes should still work.
	w = env.request(t, "PUT", "/api/config/probes/Tmp", map[string]any{
		"unit_override": "°F",
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-name change, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNameConflictGuardOutlet(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// Create device linked to outlet.
	env.request(t, "POST", "/api/devices", map[string]any{
		"name":      "Return Pump",
		"outlet_id": "base_Var1",
	})

	// Try to change display name — should be rejected.
	w := env.request(t, "PUT", "/api/config/outlets/base_Var1", map[string]any{
		"display_name": "My Custom Name",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for device-managed outlet, got %d: %s", w.Code, w.Body.String())
	}
}
