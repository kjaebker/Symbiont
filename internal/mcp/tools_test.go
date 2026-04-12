package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupMockAPI(t *testing.T) (*httptest.Server, *cli.APIClient) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/probes", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"probes": []map[string]any{
				{"name": "Tmp", "value": 78.2, "unit": "F", "status": "normal", "ts": "2026-03-22T10:00:00Z"},
				{"name": "pH", "value": 8.1, "unit": "pH", "status": "warning", "ts": "2026-03-22T10:00:00Z"},
			},
			"polled_at": "2026-03-22T10:00:00Z",
		})
	})

	mux.HandleFunc("GET /api/probes/{name}/history", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "Unknown" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "probe not found", "code": "not_found"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"probe":    name,
			"from":     "2026-03-21T10:00:00Z",
			"to":       "2026-03-22T10:00:00Z",
			"interval": "5m",
			"data": []map[string]any{
				{"ts": "2026-03-21T10:00:00Z", "value": 78.0},
				{"ts": "2026-03-21T10:05:00Z", "value": 78.1},
			},
		})
	})

	mux.HandleFunc("GET /api/outlets", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"outlets": []map[string]any{
				{"id": "base_Var1", "name": "Return", "state": "AON", "type": "outlet"},
				{"id": "base_Var2", "name": "Skimmer", "state": "OFF", "type": "outlet"},
			},
		})
	})

	mux.HandleFunc("PUT /api/outlets/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var body struct {
			State string `json:"state"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(map[string]any{
			"id": id, "name": "Return", "state": body.State, "logged_at": "2026-03-22T10:00:00Z",
		})
	})

	mux.HandleFunc("GET /api/outlets/events", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{}})
	})

	mux.HandleFunc("GET /api/alerts", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"rules": []map[string]any{
				{
					"id": float64(10), "probe_name": "Tmp", "condition": "above",
					"threshold_high": 82.0, "severity": "warning", "cooldown_minutes": 15, "enabled": true,
				},
			},
		})
	})

	mux.HandleFunc("POST /api/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": 11, "probe_name": "Tmp", "condition": "above", "threshold_high": 82.0,
			"severity": "warning", "cooldown_minutes": 15, "enabled": true,
		})
	})

	mux.HandleFunc("PUT /api/alerts/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		body["id"] = 10
		json.NewEncoder(w).Encode(body)
	})

	mux.HandleFunc("DELETE /api/alerts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/alerts/events", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"events": []map[string]any{}})
	})

	mux.HandleFunc("GET /api/system/log", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"lines": []map[string]any{}})
	})

	mux.HandleFunc("GET /api/devices", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"devices": []map[string]any{
				{"id": 1, "name": "Return Pump", "device_type": "pump"},
			},
		})
	})

	mux.HandleFunc("GET /api/system", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"controller": map[string]string{
				"serial": "AC5:12345", "hostname": "apex", "software": "5.10",
			},
			"poll_ok":       true,
			"last_poll":     "2026-03-22T10:00:00Z",
			"poll_interval": "10s",
		})
	})

	mux.HandleFunc("GET /api/livestock", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"livestock": []map[string]any{
				{"id": 1, "name": "Ocellaris Clownfish", "type": "fish", "quantity": 2, "status": "healthy"},
				{"id": 2, "name": "Acropora millepora", "type": "coral", "quantity": 1, "status": "sick"},
			},
		})
	})

	mux.HandleFunc("POST /api/livestock", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": 3, "name": "Test Fish", "type": "fish", "quantity": 1, "status": "healthy",
		})
	})

	mux.HandleFunc("GET /api/livestock/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "name": "Ocellaris Clownfish", "type": "fish", "quantity": 2, "status": "healthy",
		})
	})

	mux.HandleFunc("PUT /api/livestock/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "name": "Ocellaris Clownfish", "type": "fish", "quantity": 2, "status": "sick",
		})
	})

	mux.HandleFunc("GET /api/measurements/parameters", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"parameters": []map[string]any{
				{"id": 1, "name": "Alkalinity", "canonical_unit": "dKH"},
				{"id": 2, "name": "Calcium", "canonical_unit": "ppm"},
				{"id": 3, "name": "Magnesium", "canonical_unit": "ppm"},
			},
		})
	})

	mux.HandleFunc("GET /api/measurements", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"measurements": []map[string]any{
				{"id": 1, "measured_at": "2026-03-22T10:00:00Z", "parameter": "Alkalinity", "canonical_unit": "dKH", "value": 9.5},
			},
		})
	})

	mux.HandleFunc("POST /api/measurements", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": 2, "measured_at": "2026-03-22T10:00:00Z", "parameter": "Alkalinity", "canonical_unit": "dKH", "value": 9.5,
		})
	})

	mux.HandleFunc("DELETE /api/measurements/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/livestock/{id}/observations", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"observations": []map[string]any{
				{"id": 1, "ts": "2026-03-22T10:00:00Z", "status": "healthy", "note": "Looking good"},
				{"id": 2, "ts": "2026-03-23T10:00:00Z", "status": "sick", "note": "Possible ich"},
			},
		})
	})

	mux.HandleFunc("POST /api/livestock/{id}/observations", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 3, "ts": "2026-03-24T10:00:00Z", "status": "sick", "note": "Still showing spots"})
	})

	mux.HandleFunc("GET /api/feed", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"name": 0, "active": false})
	})

	mux.HandleFunc("PUT /api/feed", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(body)
	})

	mux.HandleFunc("GET /api/journal", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"entries": []map[string]any{
				{"id": 1, "ts": "2026-03-22T10:00:00Z", "category": "maintenance", "title": "Water change", "source": "manual"},
			},
		})
	})

	mux.HandleFunc("POST /api/journal", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": 2, "ts": "2026-03-22T10:00:00Z", "category": "observation", "title": "pH stable", "source": "ai",
		})
	})

	mux.HandleFunc("GET /api/journal/templates", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"templates": map[string]any{
				"maintenance": []map[string]any{
					{"title": "Water Change", "body": "Changed X% water"},
					{"title": "Skimmer Cleaned"},
				},
				"observation": []map[string]any{
					{"title": "Parameter Check"},
				},
			},
		})
	})

	mux.HandleFunc("GET /api/tank/profile", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"display": map[string]any{
				"section": "display", "volume_gallons": 90.0, "tank_type": "reef",
				"setup_date": "2023-01-15",
			},
			"sump": map[string]any{
				"section": "sump", "volume_gallons": 30.0,
			},
		})
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := cli.NewAPIClient(ts.URL, "test-token")
	return ts, client
}

// handlerMap maps tool names to their handlers for testing.
type handlerMap map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

func setupTools(t *testing.T, client *cli.APIClient) handlerMap {
	t.Helper()
	handlers := make(handlerMap)
	handlers["get_current_parameters"] = getCurrentParametersHandler(client)
	handlers["get_probe_history"] = getProbeHistoryHandler(client)
	handlers["get_outlet_states"] = getOutletStatesHandler(client)
	handlers["control_outlet"] = controlOutletHandler(client)
	handlers["get_outlet_event_log"] = getOutletEventLogHandler(client)
	handlers["get_alert_rules"] = getAlertRulesHandler(client)
	handlers["get_alert_events"] = getAlertEventsHandler(client)
	handlers["create_alert_rule"] = createAlertRuleHandler(client)
	handlers["update_alert_rule"] = updateAlertRuleHandler(client)
	handlers["delete_alert_rule"] = deleteAlertRuleHandler(client)
	handlers["get_system_status"] = getSystemStatusHandler(client)
	handlers["get_system_log"] = getSystemLogHandler(client)
	handlers["get_devices"] = getDevicesHandler(client)
	handlers["summarize_tank_health"] = summarizeTankHealthHandler(client)
	handlers["get_measurement_parameters"] = getMeasurementParametersHandler(client)
	handlers["get_measurements"] = getMeasurementsHandler(client)
	handlers["add_measurement"] = addMeasurementHandler(client)
	handlers["delete_measurement"] = deleteMeasurementHandler(client)
	handlers["get_livestock"] = getLivestockHandler(client)
	handlers["add_livestock"] = addLivestockHandler(client)
	handlers["update_livestock"] = updateLivestockHandler(client)
	handlers["get_livestock_observations"] = getLivestockObservationsHandler(client)
	handlers["add_livestock_observation"] = addLivestockObservationHandler(client)
	handlers["get_feed_mode"] = getFeedModeHandler(client)
	handlers["set_feed_mode"] = setFeedModeHandler(client)
	handlers["get_journal_entries"] = getJournalEntriesHandler(client)
	handlers["add_journal_entry"] = addJournalEntryHandler(client)
	handlers["get_journal_templates"] = getJournalTemplatesHandler(client)
	handlers["get_tank_profile"] = getTankProfileHandler(client)
	return handlers
}

func callTool(t *testing.T, handlers handlerMap, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	handler, ok := handlers[name]
	if !ok {
		t.Fatalf("unknown tool: %s", name)
	}
	if args == nil {
		args = map[string]any{}
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool %s error: %v", name, err)
	}
	return result
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestGetCurrentParameters(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_current_parameters", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Tmp") || !strings.Contains(text, "78.2") {
		t.Fatalf("expected probe data in result, got: %s", text)
	}
}

func TestGetProbeHistory(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_probe_history", map[string]any{"name": "Tmp"})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "78") {
		t.Fatalf("expected history data, got: %s", text)
	}
}

func TestGetProbeHistoryNotFound(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_probe_history", map[string]any{"name": "Unknown"})
	if !result.IsError {
		t.Fatal("expected error for unknown probe")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "not found") {
		t.Fatalf("expected 'not found' error, got: %s", text)
	}
}

func TestGetOutletStates(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_outlet_states", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Return") || !strings.Contains(text, "AON") {
		t.Fatalf("expected outlet data, got: %s", text)
	}
}

func TestControlOutlet(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "control_outlet", map[string]any{"id": "base_Var1", "state": "OFF"})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "OFF") {
		t.Fatalf("expected OFF in result, got: %s", text)
	}
}

func TestControlOutletInvalidState(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "control_outlet", map[string]any{"id": "base_Var1", "state": "INVALID"})
	if !result.IsError {
		t.Fatal("expected error for invalid state")
	}
}

func TestGetAlertRules(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_alert_rules", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestGetSystemStatus(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_system_status", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "AC5:12345") {
		t.Fatalf("expected serial in result, got: %s", text)
	}
}

func TestSummarizeTankHealth(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "summarize_tank_health", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)

	// Should contain synthesized health data.
	if !strings.Contains(text, "all_normal") {
		t.Fatalf("expected 'all_normal' in health summary, got: %s", text)
	}
	if !strings.Contains(text, "warnings") {
		t.Fatalf("expected 'warnings' in health summary, got: %s", text)
	}
	// pH is in warning state.
	if !strings.Contains(text, "pH") {
		t.Fatalf("expected 'pH' in warnings, got: %s", text)
	}
	// Livestock summary should be present.
	if !strings.Contains(text, "livestock") {
		t.Fatalf("expected 'livestock' in health summary, got: %s", text)
	}
}

func TestToolWithAPIDown(t *testing.T) {
	// Client pointing to a closed server.
	client := cli.NewAPIClient("http://127.0.0.1:1", "test-token")
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_current_parameters", nil)
	if !result.IsError {
		t.Fatal("expected error when API is down")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Cannot reach") {
		t.Fatalf("expected 'Cannot reach' error, got: %s", text)
	}
}

func TestGetOutletEventLog(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_outlet_event_log", map[string]any{"limit": float64(10)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestGetAlertEvents(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_alert_events", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "events") {
		t.Fatalf("expected 'events' in result, got: %s", text)
	}
}

func TestGetAlertEventsActiveOnly(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_alert_events", map[string]any{"active_only": "true", "limit": float64(5)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestGetSystemLog(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_system_log", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "lines") {
		t.Fatalf("expected 'lines' in result, got: %s", text)
	}
}

func TestGetDevices(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_devices", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Return Pump") {
		t.Fatalf("expected device in result, got: %s", text)
	}
}

func TestGetLivestock(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_livestock", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Ocellaris Clownfish") {
		t.Fatalf("expected livestock in result, got: %s", text)
	}
}

func TestGetLivestockFiltered(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_livestock", map[string]any{"type": "fish"})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestAddLivestock(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_livestock", map[string]any{
		"name": "Test Fish",
		"type": "fish",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestUpdateLivestock(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "update_livestock", map[string]any{
		"id":     float64(1),
		"status": "sick",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestSummarizeTankHealthIncludesLivestock(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "summarize_tank_health", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "livestock") {
		t.Fatalf("expected 'livestock' in health summary, got: %s", text)
	}
	if !strings.Contains(text, "by_type") {
		t.Fatalf("expected 'by_type' in livestock summary, got: %s", text)
	}
}

func TestGetMeasurementParameters(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_measurement_parameters", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "Alkalinity") {
		t.Fatalf("expected 'Alkalinity' in result, got: %s", text)
	}
	if !strings.Contains(text, "dKH") {
		t.Fatalf("expected unit 'dKH' in result, got: %s", text)
	}
}

func TestGetMeasurements(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_measurements", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "measurements") {
		t.Fatalf("expected 'measurements' in result, got: %s", text)
	}
}

func TestAddMeasurement(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_measurement", map[string]any{
		"parameter": "Alkalinity",
		"value":     float64(9.5),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestDeleteMeasurement(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "delete_measurement", map[string]any{"id": float64(1)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "deleted") {
		t.Fatalf("expected 'deleted' in result, got: %s", text)
	}
}

func TestDeleteMeasurementMissingID(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "delete_measurement", nil)
	if !result.IsError {
		t.Fatal("expected error when id is missing")
	}
}

func TestGetLivestockObservations(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_livestock_observations", map[string]any{"id": float64(1)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "observations") {
		t.Fatalf("expected 'observations' in result, got: %s", text)
	}
	if !strings.Contains(text, "Looking good") {
		t.Fatalf("expected observation note in result, got: %s", text)
	}
}

func TestGetLivestockObservationsMissingID(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_livestock_observations", nil)
	if !result.IsError {
		t.Fatal("expected error when id is missing")
	}
}

func TestAddLivestockObservation(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_livestock_observation", map[string]any{
		"id":   float64(1),
		"note": "Still showing spots",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestAddLivestockObservationWithStatus(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_livestock_observation", map[string]any{
		"id":     float64(1),
		"status": "sick",
		"note":   "Clamped fins",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestAddLivestockObservationMissingFields(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_livestock_observation", map[string]any{"id": float64(1)})
	if !result.IsError {
		t.Fatal("expected error when neither status nor note provided")
	}
	if !strings.Contains(resultText(t, result), "required") {
		t.Fatalf("expected 'required' in error, got: %s", resultText(t, result))
	}
}

func TestGetFeedMode(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_feed_mode", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "active") {
		t.Fatalf("expected 'active' in result, got: %s", text)
	}
}

func TestSetFeedMode(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "set_feed_mode", map[string]any{"name": float64(1)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestSetFeedModeCancel(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "set_feed_mode", map[string]any{"name": float64(0)})
	if result.IsError {
		t.Fatalf("unexpected error cancelling feed mode: %s", resultText(t, result))
	}
}

func TestSetFeedModeInvalidName(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "set_feed_mode", map[string]any{"name": float64(5)})
	if !result.IsError {
		t.Fatal("expected error for out-of-range feed mode name")
	}
}

func TestSetFeedModeMissingName(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "set_feed_mode", nil)
	if !result.IsError {
		t.Fatal("expected error when name is missing")
	}
}

func TestCreateAlertRule(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "create_alert_rule", map[string]any{
		"probe_name":      "Tmp",
		"condition":       "above",
		"threshold_high":  float64(82),
		"severity":        "warning",
		"cooldown_minutes": float64(15),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "probe_name") {
		t.Fatalf("expected 'probe_name' in result, got: %s", text)
	}
}

func TestCreateAlertRuleMissingProbe(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "create_alert_rule", map[string]any{"condition": "above"})
	if !result.IsError {
		t.Fatal("expected error when probe_name is missing")
	}
}

func TestUpdateAlertRule(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "update_alert_rule", map[string]any{
		"id":       float64(10),
		"severity": "critical",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestUpdateAlertRuleNotFound(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "update_alert_rule", map[string]any{
		"id":       float64(999),
		"severity": "critical",
	})
	if !result.IsError {
		t.Fatal("expected error for non-existent rule ID")
	}
}

func TestDeleteAlertRule(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "delete_alert_rule", map[string]any{"id": float64(10)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "deleted") {
		t.Fatalf("expected 'deleted' in result, got: %s", text)
	}
}

func TestDeleteAlertRuleMissingID(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "delete_alert_rule", nil)
	if !result.IsError {
		t.Fatal("expected error when id is missing")
	}
}

func TestGetJournalEntries(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_journal_entries", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "entries") {
		t.Fatalf("expected 'entries' in result, got: %s", text)
	}
}

func TestAddJournalEntry(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "add_journal_entry", map[string]any{
		"title":    "pH stable this week",
		"category": "observation",
		"sentiment": "good",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
}

func TestGetJournalTemplates(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_journal_templates", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "templates") {
		t.Fatalf("expected 'templates' in result, got: %s", text)
	}
	if !strings.Contains(text, "Water Change") {
		t.Fatalf("expected template title in result, got: %s", text)
	}
}

func TestGetTankProfile(t *testing.T) {
	_, client := setupMockAPI(t)
	handlers := setupTools(t, client)

	result := callTool(t, handlers, "get_tank_profile", nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "display") {
		t.Fatalf("expected 'display' section in result, got: %s", text)
	}
	if !strings.Contains(text, "90") {
		t.Fatalf("expected tank volume in result, got: %s", text)
	}
}
