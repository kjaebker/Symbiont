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
		json.NewEncoder(w).Encode(map[string]any{"alerts": []map[string]any{}})
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
	handlers["get_system_status"] = getSystemStatusHandler(client)
	handlers["summarize_tank_health"] = summarizeTankHealthHandler(client)
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
