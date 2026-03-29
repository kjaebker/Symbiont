package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools adds all Symbiont tools to the MCP server.
func RegisterTools(s *server.MCPServer, client *cli.APIClient) {
	s.AddTool(getCurrentParametersTool(), getCurrentParametersHandler(client))
	s.AddTool(getProbeHistoryTool(), getProbeHistoryHandler(client))
	s.AddTool(getOutletStatesTool(), getOutletStatesHandler(client))
	s.AddTool(controlOutletTool(), controlOutletHandler(client))
	s.AddTool(getOutletEventLogTool(), getOutletEventLogHandler(client))
	s.AddTool(getAlertRulesTool(), getAlertRulesHandler(client))
	s.AddTool(getAlertEventsTool(), getAlertEventsHandler(client))
	s.AddTool(getSystemStatusTool(), getSystemStatusHandler(client))
	s.AddTool(getSystemLogTool(), getSystemLogHandler(client))
	s.AddTool(getDevicesTool(), getDevicesHandler(client))
	s.AddTool(summarizeTankHealthTool(), summarizeTankHealthHandler(client))
}

func apiCall(client *cli.APIClient, path string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Get(ctx, path, result)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(msg)},
		IsError: true,
	}
}

// --- get_current_parameters ---

func getCurrentParametersTool() mcp.Tool {
	return mcp.NewTool("get_current_parameters",
		mcp.WithDescription("Get the current reading for all aquarium probes — temperature, pH, ORP, salinity, and any others connected to the Apex. Returns current value, unit, status (normal/warning/critical), and timestamp of last reading."),
	)
}

func getCurrentParametersHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(client, "/api/probes", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_probe_history ---

func getProbeHistoryTool() mcp.Tool {
	return mcp.NewTool("get_probe_history",
		mcp.WithDescription("Get time-series history for a specific probe. Useful for analyzing trends, correlating parameter changes with events, or understanding patterns over time."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Probe name exactly as returned by get_current_parameters"),
		),
		mcp.WithString("from",
			mcp.Description("ISO 8601 start time, default 24 hours ago"),
		),
		mcp.WithString("to",
			mcp.Description("ISO 8601 end time, default now"),
		),
		mcp.WithString("interval",
			mcp.Description("Bucket size: 10s, 1m, 5m, 15m, 1h, 1d — default auto"),
		),
	)
}

func getProbeHistoryHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError("Parameter 'name' is required"), nil
		}

		path := "/api/probes/" + name + "/history?"
		if v := request.GetString("from", ""); v != "" {
			path += "from=" + v + "&"
		}
		if v := request.GetString("to", ""); v != "" {
			path += "to=" + v + "&"
		}
		if v := request.GetString("interval", ""); v != "" {
			path += "interval=" + v + "&"
		}

		var resp any
		if err := apiCall(client, path, &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Probe '%s' not found or no data in range", name)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_outlet_states ---

func getOutletStatesTool() mcp.Tool {
	return mcp.NewTool("get_outlet_states",
		mcp.WithDescription("Get the current state of all outlets controlled by the Apex. State values: ON/OFF (manual override), AON/AOF (auto mode, program running). Includes outlet type and intensity."),
	)
}

func getOutletStatesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(client, "/api/outlets", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- control_outlet ---

func controlOutletTool() mcp.Tool {
	return mcp.NewTool("control_outlet",
		mcp.WithDescription("Set an outlet to ON, OFF, or AUTO. Use the outlet ID from get_outlet_states. AUTO returns the outlet to program control. Use with care — this directly controls aquarium equipment."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Outlet ID from get_outlet_states"),
		),
		mcp.WithString("state",
			mcp.Required(),
			mcp.Description("Desired state: ON, OFF, or AUTO"),
			mcp.Enum("ON", "OFF", "AUTO"),
		),
	)
}

func controlOutletHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return toolError("Parameter 'id' is required"), nil
		}
		state, err := request.RequireString("state")
		if err != nil {
			return toolError("Parameter 'state' is required"), nil
		}

		state = strings.ToUpper(state)
		if state != "ON" && state != "OFF" && state != "AUTO" {
			return toolError("Invalid outlet state. Must be ON, OFF, or AUTO"), nil
		}

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := client.Put(ctx, "/api/outlets/"+id, map[string]string{"state": state}, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to set outlet: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_outlet_event_log ---

func getOutletEventLogTool() mcp.Tool {
	return mcp.NewTool("get_outlet_event_log",
		mcp.WithDescription("Get a log of recent outlet state changes, including who or what made each change (ui, cli, mcp, api) and what the state changed from and to. Useful for understanding what happened in the tank over time."),
		mcp.WithString("outlet_id",
			mcp.Description("Filter to specific outlet. Omit for all outlets."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max events to return, default 20, max 100"),
		),
	)
}

func getOutletEventLogHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/outlets/events?"
		if v := request.GetString("outlet_id", ""); v != "" {
			path += "outlet_id=" + v + "&"
		}
		limit := request.GetInt("limit", 20)
		path += fmt.Sprintf("limit=%d&", limit)

		var resp any
		if err := apiCall(client, path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_alert_rules ---

func getAlertRulesTool() mcp.Tool {
	return mcp.NewTool("get_alert_rules",
		mcp.WithDescription("Get all configured alert rules — the thresholds set for each probe that trigger notifications when breached. Useful for understanding what parameter ranges are considered normal or concerning."),
	)
}

func getAlertRulesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(client, "/api/alerts", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_system_status ---

func getSystemStatusTool() mcp.Tool {
	return mcp.NewTool("get_system_status",
		mcp.WithDescription("Get Apex controller information (firmware, serial number) and Symbiont system health (last poll time, whether polling is working, database sizes). Use to confirm the system is functioning normally."),
	)
}

func getSystemStatusHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(client, "/api/system", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- summarize_tank_health ---

func summarizeTankHealthTool() mcp.Tool {
	return mcp.NewTool("summarize_tank_health",
		mcp.WithDescription("Get a comprehensive health snapshot of the aquarium — all current parameters with status, outlet states, and system health. Best starting point for a general tank status check."),
	)
}

func summarizeTankHealthHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		type probeEntry struct {
			Name   string  `json:"name"`
			Value  float64 `json:"value"`
			Unit   string  `json:"unit"`
			Status string  `json:"status"`
		}
		type outletEntry struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
			Type  string `json:"type"`
		}

		var (
			probesResp  struct{ Probes []probeEntry `json:"probes"` }
			outletsResp struct{ Outlets []outletEntry `json:"outlets"` }
			systemResp  struct {
				PollOK   bool   `json:"poll_ok"`
				LastPoll string `json:"last_poll"`
			}
			errs [3]error
			wg   sync.WaitGroup
		)

		wg.Add(3)
		go func() {
			defer wg.Done()
			errs[0] = apiCall(client, "/api/probes", &probesResp)
		}()
		go func() {
			defer wg.Done()
			errs[1] = apiCall(client, "/api/outlets", &outletsResp)
		}()
		go func() {
			defer wg.Done()
			errs[2] = apiCall(client, "/api/system", &systemResp)
		}()
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
			}
		}

		// Synthesize health summary.
		var warnings, critical []string
		allNormal := true
		for _, p := range probesResp.Probes {
			switch p.Status {
			case "warning":
				warnings = append(warnings, p.Name)
				allNormal = false
			case "critical":
				critical = append(critical, p.Name)
				allNormal = false
			}
		}

		var onCount, offCount, autoCount int
		for _, o := range outletsResp.Outlets {
			switch strings.ToUpper(o.State) {
			case "ON", "AON":
				onCount++
			case "OFF", "AOF":
				offCount++
			default:
				autoCount++
			}
		}

		summary := map[string]any{
			"system_ok":    systemResp.PollOK,
			"poll_ok":      systemResp.PollOK,
			"last_poll_ts": systemResp.LastPoll,
			"parameters": map[string]any{
				"all_normal": allNormal,
				"warnings":   warnings,
				"critical":   critical,
				"probes":     probesResp.Probes,
			},
			"outlets": map[string]any{
				"total":   len(outletsResp.Outlets),
				"on":      onCount,
				"off":     offCount,
				"auto":    autoCount,
				"outlets": outletsResp.Outlets,
			},
		}

		return jsonResult(summary)
	}
}

// --- get_alert_events ---

func getAlertEventsTool() mcp.Tool {
	return mcp.NewTool("get_alert_events",
		mcp.WithDescription("Get recent alert trigger events — when thresholds were breached, what parameter was affected, severity, peak value, and whether the alert has since cleared. Useful for understanding recent parameter problems."),
		mcp.WithNumber("limit",
			mcp.Description("Max events to return, default 20, max 100"),
		),
		mcp.WithString("active_only",
			mcp.Description("Set to 'true' to return only uncleared (still active) alerts"),
		),
	)
}

func getAlertEventsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/alerts/events?"
		limit := request.GetInt("limit", 20)
		path += fmt.Sprintf("limit=%d&", limit)
		if request.GetString("active_only", "") == "true" {
			path += "active_only=true&"
		}

		var resp any
		if err := apiCall(client, path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_system_log ---

func getSystemLogTool() mcp.Tool {
	return mcp.NewTool("get_system_log",
		mcp.WithDescription("Get recent structured log entries from the Symbiont API and poller services. Useful for diagnosing errors, checking poll failures, or understanding recent system activity."),
		mcp.WithNumber("limit",
			mcp.Description("Max log lines to return, default 50, max 500"),
		),
		mcp.WithString("service",
			mcp.Description("Filter by service: api or poller. Omit for both."),
			mcp.Enum("api", "poller"),
		),
	)
}

func getSystemLogHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := request.GetInt("limit", 50)
		path := fmt.Sprintf("/api/system/log?limit=%d&", limit)
		if v := request.GetString("service", ""); v != "" {
			path += "service=" + v + "&"
		}

		var resp any
		if err := apiCall(client, path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_devices ---

func getDevicesTool() mcp.Tool {
	return mcp.NewTool("get_devices",
		mcp.WithDescription("Get the list of aquarium devices (equipment) configured in Symbiont — pumps, lights, skimmers, heaters, etc. Each device may have an associated outlet and probes. Useful for understanding what equipment is in the tank and how it relates to outlets and sensor readings."),
	)
}

func getDevicesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(client, "/api/devices", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
