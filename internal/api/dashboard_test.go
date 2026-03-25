package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kjaebker/symbiont/internal/db"
)

func TestDashboardCRUD(t *testing.T) {
	env := setupTestEnv(t)

	// GET — empty.
	w := env.request(t, "GET", "/api/dashboard", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct{ Items []db.DashboardItem }
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(listResp.Items))
	}

	// POST — add probe.
	w = env.request(t, "POST", "/api/dashboard", map[string]any{
		"item_type":    "probe",
		"reference_id": "Tmp",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// POST — add separator.
	w = env.request(t, "POST", "/api/dashboard", map[string]any{
		"item_type": "separator",
		"label":     "My Section",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// GET — should have 2 items.
	w = env.request(t, "GET", "/api/dashboard", nil)
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(listResp.Items))
	}

	// PUT — replace entire layout.
	w = env.request(t, "PUT", "/api/dashboard", map[string]any{
		"items": []map[string]any{
			{"item_type": "separator", "label": "Telemetry"},
			{"item_type": "probe", "reference_id": "Tmp"},
			{"item_type": "probe", "reference_id": "pH"},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var replaceResp struct{ Items []db.DashboardItem }
	json.NewDecoder(w.Body).Decode(&replaceResp)
	if len(replaceResp.Items) != 3 {
		t.Fatalf("expected 3 items after replace, got %d", len(replaceResp.Items))
	}
	if replaceResp.Items[0].SortOrder != 1 || replaceResp.Items[2].SortOrder != 3 {
		t.Errorf("unexpected sort orders: %d, %d", replaceResp.Items[0].SortOrder, replaceResp.Items[2].SortOrder)
	}

	// DELETE — remove last item.
	lastID := replaceResp.Items[2].ID
	w = env.request(t, "DELETE", "/api/dashboard/"+intToStr(int(lastID)), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify removal.
	w = env.request(t, "GET", "/api/dashboard", nil)
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Items) != 2 {
		t.Fatalf("expected 2 items after delete, got %d", len(listResp.Items))
	}
}

func TestDashboardValidation(t *testing.T) {
	env := setupTestEnv(t)

	// Invalid item_type.
	w := env.request(t, "POST", "/api/dashboard", map[string]any{
		"item_type":    "invalid",
		"reference_id": "x",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}

	// Separator without label.
	w = env.request(t, "POST", "/api/dashboard", map[string]any{
		"item_type": "separator",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for separator without label, got %d", w.Code)
	}

	// Probe without reference_id.
	w = env.request(t, "POST", "/api/dashboard", map[string]any{
		"item_type": "probe",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for probe without reference_id, got %d", w.Code)
	}

	// Duplicate ref in PUT.
	w = env.request(t, "PUT", "/api/dashboard", map[string]any{
		"items": []map[string]any{
			{"item_type": "probe", "reference_id": "Tmp"},
			{"item_type": "probe", "reference_id": "Tmp"},
		},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate refs, got %d", w.Code)
	}
}

func TestDeviceCreateAddsToDashboard(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// Create device.
	w := env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Test Pump",
		"device_type": "pump",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Dashboard should have the device.
	w = env.request(t, "GET", "/api/dashboard", nil)
	var resp struct{ Items []db.DashboardItem }
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 dashboard item, got %d", len(resp.Items))
	}
	if resp.Items[0].ItemType != "device" {
		t.Errorf("expected device type, got %q", resp.Items[0].ItemType)
	}
}

func TestDeviceDeleteRemovesFromDashboard(t *testing.T) {
	env := setupTestEnv(t)
	seedTestData(t, env.duck)

	// Create device (auto-adds to dashboard).
	env.request(t, "POST", "/api/devices", map[string]any{
		"name":        "Test Pump",
		"device_type": "pump",
	})

	// Delete device.
	env.request(t, "DELETE", "/api/devices/1", nil)

	// Dashboard should be empty.
	w := env.request(t, "GET", "/api/dashboard", nil)
	var resp struct{ Items []db.DashboardItem }
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 dashboard items after delete, got %d", len(resp.Items))
	}
}
