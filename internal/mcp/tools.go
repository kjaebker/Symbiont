package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
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
	s.AddTool(createAlertRuleTool(), createAlertRuleHandler(client))
	s.AddTool(updateAlertRuleTool(), updateAlertRuleHandler(client))
	s.AddTool(deleteAlertRuleTool(), deleteAlertRuleHandler(client))
	s.AddTool(getSystemStatusTool(), getSystemStatusHandler(client))
	s.AddTool(getSystemLogTool(), getSystemLogHandler(client))
	s.AddTool(getDevicesTool(), getDevicesHandler(client))
	s.AddTool(summarizeTankHealthTool(), summarizeTankHealthHandler(client))
	s.AddTool(getMeasurementParametersTool(), getMeasurementParametersHandler(client))
	s.AddTool(getMeasurementsTool(), getMeasurementsHandler(client))
	s.AddTool(addMeasurementTool(), addMeasurementHandler(client))
	s.AddTool(deleteMeasurementTool(), deleteMeasurementHandler(client))
	s.AddTool(getLivestockTool(), getLivestockHandler(client))
	s.AddTool(addLivestockTool(), addLivestockHandler(client))
	s.AddTool(updateLivestockTool(), updateLivestockHandler(client))
	s.AddTool(getLivestockObservationsTool(), getLivestockObservationsHandler(client))
	s.AddTool(addLivestockObservationTool(), addLivestockObservationHandler(client))
	s.AddTool(getLivestockImageTool(), getLivestockImageHandler(client))
	s.AddTool(getFeedModeTool(), getFeedModeHandler(client))
	s.AddTool(setFeedModeTool(), setFeedModeHandler(client))
	s.AddTool(getJournalEntriesTool(), getJournalEntriesHandler(client))
	s.AddTool(addJournalEntryTool(), addJournalEntryHandler(client))
	s.AddTool(getJournalTemplatesTool(), getJournalTemplatesHandler(client))
	s.AddTool(getTankProfileTool(), getTankProfileHandler(client))
	s.AddTool(getAgentContextTool(), getAgentContextHandler(client))
	s.AddTool(listSkillsTool(), listSkillsHandler(client))
	s.AddTool(getSkillTool(), getSkillHandler(client))
	s.AddTool(listProbeConfigsTool(), listProbeConfigsHandler(client))
	s.AddTool(updateProbeConfigTool(), updateProbeConfigHandler(client))
}

type contextKeyClient struct{}

// WithLoopbackClient stores a per-request loopback API client in ctx.
// Called by the MCP HTTP handler so each tool invocation uses the caller's token.
func WithLoopbackClient(ctx context.Context, client *cli.APIClient) context.Context {
	return context.WithValue(ctx, contextKeyClient{}, client)
}

// clientFromCtx returns the per-request loopback client from ctx,
// falling back to the static client (used in stdio mode).
func clientFromCtx(ctx context.Context, fallback *cli.APIClient) *cli.APIClient {
	if c, ok := ctx.Value(contextKeyClient{}).(*cli.APIClient); ok {
		return c
	}
	return fallback
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
		if err := apiCall(clientFromCtx(ctx, client),"/api/probes", &resp); err != nil {
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

		u := &url.URL{Path: "/api/probes/" + url.PathEscape(name) + "/history"}
		q := url.Values{}
		if v := request.GetString("from", ""); v != "" {
			q.Set("from", v)
		}
		if v := request.GetString("to", ""); v != "" {
			q.Set("to", v)
		}
		if v := request.GetString("interval", ""); v != "" {
			q.Set("interval", v)
		}
		u.RawQuery = q.Encode()
		path := u.String()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client),"/api/outlets", &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Put(ctx,"/api/outlets/"+id, map[string]string{"state": state}, &resp); err != nil {
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
			mcp.Description("Filter to a specific outlet by its ID. Omit for all outlets."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max events to return, default 20, max 100"),
		),
	)
}

func getOutletEventLogHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{"kind": {"outlet_changed"}}
		if v := request.GetString("outlet_id", ""); v != "" {
			q.Set("correlation_id", v)
		}
		q.Set("limit", fmt.Sprintf("%d", request.GetInt("limit", 20)))
		path := "/api/events?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client),"/api/alerts", &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client),"/api/system", &resp); err != nil {
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

		type livestockEntry struct {
			ID       int64   `json:"id"`
			Name     string  `json:"name"`
			Type     string  `json:"type"`
			Quantity int     `json:"quantity"`
			Status   string  `json:"status"`
			Species  *string `json:"species"`
		}

		var (
			probesResp  struct{ Probes []probeEntry `json:"probes"` }
			outletsResp struct{ Outlets []outletEntry `json:"outlets"` }
			systemResp  struct {
				PollOK   bool   `json:"poll_ok"`
				LastPoll string `json:"last_poll"`
			}
			livestockResp struct{ Livestock []livestockEntry `json:"livestock"` }
			errs          [4]error
			wg            sync.WaitGroup
		)

		wg.Add(4)
		go func() {
			defer wg.Done()
			errs[0] = apiCall(clientFromCtx(ctx, client),"/api/probes", &probesResp)
		}()
		go func() {
			defer wg.Done()
			errs[1] = apiCall(clientFromCtx(ctx, client),"/api/outlets", &outletsResp)
		}()
		go func() {
			defer wg.Done()
			errs[2] = apiCall(clientFromCtx(ctx, client),"/api/system", &systemResp)
		}()
		go func() {
			defer wg.Done()
			errs[3] = apiCall(clientFromCtx(ctx, client),"/api/livestock", &livestockResp)
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

		// Aggregate livestock by type (sum quantities) and collect non-healthy items.
		byType := map[string]int{}
		type nonHealthyEntry struct {
			ID     int64   `json:"id"`
			Name   string  `json:"name"`
			Type   string  `json:"type"`
			Status string  `json:"status"`
		}
		var nonHealthy []nonHealthyEntry
		for _, item := range livestockResp.Livestock {
			byType[item.Type] += item.Quantity
			if item.Status != "healthy" {
				nonHealthy = append(nonHealthy, nonHealthyEntry{
					ID:     item.ID,
					Name:   item.Name,
					Type:   item.Type,
					Status: item.Status,
				})
			}
		}
		if nonHealthy == nil {
			nonHealthy = []nonHealthyEntry{}
		}

		totalLivestock := 0
		for _, q := range byType {
			totalLivestock += q
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
			"livestock": map[string]any{
				"total":       totalLivestock,
				"by_type":     byType,
				"non_healthy": nonHealthy,
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
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", request.GetInt("limit", 20)))
		if request.GetString("active_only", "") == "true" {
			q.Set("active_only", "true")
		}
		path := "/api/alerts/events?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
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
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", request.GetInt("limit", 50)))
		if v := request.GetString("service", ""); v != "" {
			q.Set("service", v)
		}
		path := "/api/system/log?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client),"/api/devices", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_measurements ---

func getMeasurementsTool() mcp.Tool {
	return mcp.NewTool("get_measurements",
		mcp.WithDescription("Get manual water chemistry measurements logged in Symbiont — alkalinity, calcium, magnesium, nitrate, phosphate, etc. Returns timestamped readings with parameter name and unit. Useful for tracking water quality trends, identifying parameter drift, or correlating chemistry changes with tank events."),
		mcp.WithString("parameter",
			mcp.Description("Filter by parameter name — e.g. 'Alkalinity', 'Magnesium', 'Calcium'. Omit for all parameters."),
		),
		mcp.WithString("from",
			mcp.Description("ISO 8601 start time — e.g. '2026-01-01T00:00:00Z'. Omit for no lower bound."),
		),
		mcp.WithString("to",
			mcp.Description("ISO 8601 end time. Omit for now."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results to return, default 200."),
		),
	)
}

func getMeasurementsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := request.GetString("parameter", ""); v != "" {
			q.Set("parameter", v)
		}
		if v := request.GetString("from", ""); v != "" {
			q.Set("from", v)
		}
		if v := request.GetString("to", ""); v != "" {
			q.Set("to", v)
		}
		if v := request.GetInt("limit", 0); v > 0 {
			q.Set("limit", fmt.Sprintf("%d", v))
		}
		path := "/api/measurements?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_measurement ---

func addMeasurementTool() mcp.Tool {
	return mcp.NewTool("add_measurement",
		mcp.WithDescription("Log a manual water chemistry measurement. Use this to record test results for alkalinity, calcium, magnesium, nitrate, phosphate, and other parameters. The parameter name must be one of the known parameters (use get_measurement_parameters to list them)."),
		mcp.WithString("parameter",
			mcp.Required(),
			mcp.Description("Parameter name — e.g. 'Alkalinity', 'Magnesium'. Must be a known parameter."),
		),
		mcp.WithNumber("value",
			mcp.Required(),
			mcp.Description("Measured value in the parameter's canonical unit (e.g. 9.5 for Alkalinity in dKH, 1350 for Magnesium in ppm)."),
		),
		mcp.WithString("measured_at",
			mcp.Description("ISO 8601 timestamp of when the measurement was taken. Defaults to now."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes — e.g. 'after 10% water change'."),
		),
	)
}

func addMeasurementHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parameter, err := request.RequireString("parameter")
		if err != nil {
			return toolError("Parameter 'parameter' is required"), nil
		}
		value := request.GetFloat("value", 0)

		body := map[string]any{
			"parameter": parameter,
			"value":     value,
		}
		if v := request.GetString("measured_at", ""); v != "" {
			body["measured_at"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx,"/api/measurements", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add measurement: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_livestock ---

func getLivestockTool() mcp.Tool {
	return mcp.NewTool("get_livestock",
		mcp.WithDescription("Get the list of aquarium livestock — fish, coral, and invertebrates. Returns name, species, type, quantity, current health status, and date added. Use to understand what lives in the tank and the health of each resident."),
		mcp.WithString("type",
			mcp.Description("Filter by type: fish, coral, invertebrate, other. Omit for all."),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by health status: healthy, sick, quarantine, deceased. Omit for all."),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
	)
}

func getLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := request.GetString("type", ""); v != "" {
			q.Set("type", v)
		}
		if v := request.GetString("status", ""); v != "" {
			q.Set("status", v)
		}
		path := "/api/livestock?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_livestock ---

func addLivestockTool() mcp.Tool {
	return mcp.NewTool("add_livestock",
		mcp.WithDescription("Add a new livestock item (fish, coral, or invertebrate) to the tank log."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Common name — e.g. 'Ocellaris Clownfish', 'Acropora millepora'"),
		),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Type of livestock"),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("species",
			mcp.Description("Scientific or species name — e.g. 'Amphiprion ocellaris'"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Number of this item (default 1)"),
		),
		mcp.WithString("status",
			mcp.Description("Initial health status (default: healthy)"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("date_added",
			mcp.Description("Date added to the tank (YYYY-MM-DD). Defaults to today if omitted."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes"),
		),
	)
}

func addLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError("Parameter 'name' is required"), nil
		}
		livestockType, err := request.RequireString("type")
		if err != nil {
			return toolError("Parameter 'type' is required"), nil
		}

		body := map[string]any{
			"name": name,
			"type": livestockType,
		}
		if v := request.GetString("species", ""); v != "" {
			body["species"] = v
		}
		if v := request.GetInt("quantity", 0); v > 0 {
			body["quantity"] = v
		}
		if v := request.GetString("status", ""); v != "" {
			body["status"] = v
		}
		if v := request.GetString("date_added", ""); v != "" {
			body["date_added"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx,"/api/livestock", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add livestock: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- update_livestock ---

func updateLivestockTool() mcp.Tool {
	return mcp.NewTool("update_livestock",
		mcp.WithDescription("Update an existing livestock item. Commonly used to record status changes — e.g. marking a fish as sick or deceased. The id must be the numeric ID from get_livestock."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithString("name",
			mcp.Description("Common name"),
		),
		mcp.WithString("type",
			mcp.Description("Type: fish, coral, invertebrate, other"),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("species",
			mcp.Description("Scientific or species name"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Quantity"),
		),
		mcp.WithString("status",
			mcp.Description("Health status"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("date_added",
			mcp.Description("Date added (YYYY-MM-DD)"),
		),
		mcp.WithString("notes",
			mcp.Description("Notes"),
		),
	)
}

func updateLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		// Fetch existing item to merge changes.
		var existing struct {
			ID        int64   `json:"id"`
			Name      string  `json:"name"`
			Species   *string `json:"species"`
			Type      string  `json:"type"`
			Quantity  int     `json:"quantity"`
			Status    string  `json:"status"`
			DateAdded *string `json:"date_added"`
			Notes     *string `json:"notes"`
		}
		if err := apiCall(clientFromCtx(ctx, client),"/api/livestock/"+idStr, &existing); err != nil {
			return toolError(fmt.Sprintf("Livestock item %s not found: %v", idStr, err)), nil
		}

		body := map[string]any{
			"name":     existing.Name,
			"type":     existing.Type,
			"quantity": existing.Quantity,

			"status":   existing.Status,
		}
		if existing.Species != nil {
			body["species"] = *existing.Species
		}
		if existing.DateAdded != nil {
			body["date_added"] = *existing.DateAdded
		}
		if existing.Notes != nil {
			body["notes"] = *existing.Notes
		}

		// Apply provided overrides.
		if v := request.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := request.GetString("type", ""); v != "" {
			body["type"] = v
		}
		if v := request.GetString("species", ""); v != "" {
			body["species"] = v
		}
		if v := request.GetInt("quantity", 0); v > 0 {
			body["quantity"] = v
		}
		if v := request.GetString("status", ""); v != "" {
			body["status"] = v
		}
		if v := request.GetString("date_added", ""); v != "" {
			body["date_added"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Put(timeoutCtx,"/api/livestock/"+idStr, body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to update livestock: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_journal_entries ---

func getJournalEntriesTool() mcp.Tool {
	return mcp.NewTool("get_journal_entries",
		mcp.WithDescription("Get the tank journal — a chronological log of observations, maintenance tasks, events, and milestones. Includes both manual entries and system-generated entries (e.g. feed mode activations). Useful for understanding tank history, correlating events with parameter changes, or drafting a status summary."),
		mcp.WithString("category",
			mcp.Description("Filter by category. Omit for all."),
			mcp.Enum("observation", "maintenance", "event", "milestone"),
		),
		mcp.WithString("sentiment",
			mcp.Description("Filter by sentiment. Omit for all."),
			mcp.Enum("good", "neutral", "bad", "critical"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max entries to return (default 50, max 500)."),
		),
	)
}

func getJournalEntriesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := request.GetString("category", ""); v != "" {
			q.Set("category", v)
		}
		if v := request.GetString("sentiment", ""); v != "" {
			q.Set("sentiment", v)
		}
		if v := request.GetFloat("limit", 0); v > 0 {
			q.Set("limit", fmt.Sprintf("%d", int(v)))
		}
		path := "/api/journal?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_journal_entry ---

func addJournalEntryTool() mcp.Tool {
	return mcp.NewTool("add_journal_entry",
		mcp.WithDescription("Add a journal entry to the tank log. Use this to record observations you make while reviewing parameters, note maintenance that was performed, log significant events, or flag concerns. Entries you create will have source 'ai' so they are distinguishable from manual entries."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short title summarizing the entry — e.g. 'Alkalinity trending low' or 'Skimmer cleaned'"),
		),
		mcp.WithString("category",
			mcp.Required(),
			mcp.Description("Category of this entry"),
			mcp.Enum("observation", "maintenance", "event", "milestone"),
		),
		mcp.WithString("sentiment",
			mcp.Description("Overall sentiment of this entry"),
			mcp.Enum("good", "neutral", "bad", "critical"),
		),
		mcp.WithString("body",
			mcp.Description("Additional details — include specific values, context, or recommendations"),
		),
	)
}

func addJournalEntryHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := request.RequireString("title")
		if err != nil {
			return toolError("Parameter 'title' is required"), nil
		}
		category, err := request.RequireString("category")
		if err != nil {
			return toolError("Parameter 'category' is required"), nil
		}

		body := map[string]any{
			"title":    title,
			"category": category,
			"source":   "ai",
		}
		if v := request.GetString("sentiment", ""); v != "" {
			body["sentiment"] = v
		}
		if v := request.GetString("body", ""); v != "" {
			body["body"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx,"/api/journal", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add journal entry: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_tank_profile ---

func getTankProfileTool() mcp.Tool {
	return mcp.NewTool("get_tank_profile",
		mcp.WithDescription("Get the physical profile of the display tank and sump — volume, dimensions, tank type, manufacturer, model, and setup date. Useful for context when answering questions about stocking levels, equipment sizing, or how long the tank has been running."),
	)
}

func getTankProfileHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/tank/profile", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_measurement_parameters ---

func getMeasurementParametersTool() mcp.Tool {
	return mcp.NewTool("get_measurement_parameters",
		mcp.WithDescription("List all known water chemistry measurement parameters (alkalinity, calcium, magnesium, etc.) with their canonical units. Use this to discover valid parameter names before calling add_measurement."),
	)
}

func getMeasurementParametersHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/measurements/parameters", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- delete_measurement ---

func deleteMeasurementTool() mcp.Tool {
	return mcp.NewTool("delete_measurement",
		mcp.WithDescription("Delete a manual measurement entry. Use to correct data entry mistakes. Use get_measurements to find the measurement ID."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Measurement ID from get_measurements"),
		),
	)
}

func deleteMeasurementHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := clientFromCtx(ctx, client).Delete(timeoutCtx,"/api/measurements/"+idStr); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Measurement %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to delete measurement: %v", err)), nil
		}
		return mcp.NewToolResultText("Measurement " + idStr + " deleted"), nil
	}
}

// --- get_livestock_observations ---

func getLivestockObservationsTool() mcp.Tool {
	return mcp.NewTool("get_livestock_observations",
		mcp.WithDescription("Get the health observation history for a specific livestock item — timestamped entries with health status and notes. Use the id from get_livestock. Useful for tracking how a fish or coral has been doing over time."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
	)
}

func getLivestockObservationsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/livestock/"+idStr+"/observations", &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_livestock_observation ---

func addLivestockObservationTool() mcp.Tool {
	return mcp.NewTool("add_livestock_observation",
		mcp.WithDescription("Log a health observation for a livestock item. Provide a status change, a note, or both. Use the id from get_livestock."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithString("status",
			mcp.Description("Health status at time of observation"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("note",
			mcp.Description("Observation note — describe what you observed"),
		),
	)
}

func addLivestockObservationHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		status := request.GetString("status", "")
		note := request.GetString("note", "")
		if status == "" && note == "" {
			return toolError("At least one of 'status' or 'note' is required"), nil
		}

		body := map[string]any{}
		if status != "" {
			body["status"] = status
		}
		if note != "" {
			body["note"] = note
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx,"/api/livestock/"+idStr+"/observations", body, &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to add observation: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_livestock_image ---

func getLivestockImageTool() mcp.Tool {
	return mcp.NewTool("get_livestock_image",
		mcp.WithDescription("Fetch the photo for a livestock item so you can visually inspect it. Returns the image as inline content. Use the id from get_livestock. Pass thumbnail=true (the default) for a fast preview, or thumbnail=false for the full-resolution image."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithBoolean("thumbnail",
			mcp.Description("Return the thumbnail instead of the full image (default: true)"),
		),
	)
}

func getLivestockImageHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		// Default thumbnail=true unless explicitly set to false.
		useThumbnail := true
		if v, ok := request.GetArguments()["thumbnail"]; ok {
			if b, ok := v.(bool); ok {
				useThumbnail = b
			}
		}

		// Fetch the livestock item to get its image_path.
		var item struct {
			Name      string  `json:"name"`
			ImagePath *string `json:"image_path"`
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := clientFromCtx(ctx, client).Get(timeoutCtx,"/api/livestock/"+idStr, &item); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		if item.ImagePath == nil {
			return mcp.NewToolResultText(fmt.Sprintf("Livestock item %s (%s) has no image.", idStr, item.Name)), nil
		}

		imagePath := *item.ImagePath
		if useThumbnail {
			imagePath = thumbPathMCP(imagePath)
		}

		data, contentType, err := clientFromCtx(ctx, client).GetBytes(timeoutCtx,"/"+imagePath)
		if err != nil {
			// Fall back to full image if thumbnail doesn't exist yet.
			if useThumbnail {
				data, contentType, err = clientFromCtx(ctx, client).GetBytes(timeoutCtx,"/"+*item.ImagePath)
			}
			if err != nil {
				return toolError(fmt.Sprintf("Failed to fetch image: %v", err)), nil
			}
		}

		if contentType == "" {
			contentType = "image/jpeg"
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Photo of %s (livestock ID %s):", item.Name, idStr)),
				mcp.NewImageContent(encoded, contentType),
			},
		}, nil
	}
}

// thumbPathMCP derives the thumbnail path from an image path, mirroring the
// naming convention used by the API server's image upload handlers.
func thumbPathMCP(imagePath string) string {
	for i := len(imagePath) - 1; i >= 0 && imagePath[i] != '/'; i-- {
		if imagePath[i] == '.' {
			return imagePath[:i] + "-thumb.jpg"
		}
	}
	return imagePath + "-thumb.jpg"
}

// --- get_feed_mode ---

func getFeedModeTool() mcp.Tool {
	return mcp.NewTool("get_feed_mode",
		mcp.WithDescription("Get the current feed mode status. Feed modes temporarily suspend certain equipment (e.g. powerheads) to allow feeding without livestock getting caught. Name values: 0=off, 1=Feed A, 2=Feed B, 3=Feed C, 4=Feed D."),
	)
}

func getFeedModeHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/feed", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- set_feed_mode ---

func setFeedModeTool() mcp.Tool {
	return mcp.NewTool("set_feed_mode",
		mcp.WithDescription("Activate or cancel a feed mode on the Apex controller. Feed modes suspend equipment during feeding. Use name 0 or active=false to cancel. Use with care — this affects equipment operation."),
		mcp.WithNumber("name",
			mcp.Required(),
			mcp.Description("Feed mode: 0=cancel, 1=Feed A, 2=Feed B, 3=Feed C, 4=Feed D"),
		),
		mcp.WithBoolean("active",
			mcp.Description("Whether to activate (true) or cancel (false) the feed mode. Defaults to true when name is 1–4."),
		),
	)
}

func setFeedModeHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use -1 sentinel so name=0 (cancel) is distinguishable from absent.
		nameFloat := request.GetFloat("name", -1)
		if nameFloat < 0 {
			return toolError("Parameter 'name' is required (0=cancel, 1–4=Feed A–D)"), nil
		}
		name := int(nameFloat)
		if name > 4 {
			return toolError("Parameter 'name' must be 0–4 (0=cancel, 1=Feed A, 2=Feed B, 3=Feed C, 4=Feed D)"), nil
		}

		active := name > 0 // default: active if a feed mode is named
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if v, ok := args["active"]; ok {
				if b, ok := v.(bool); ok {
					active = b
				}
			}
		}

		body := map[string]any{
			"name":   name,
			"active": active,
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Put(timeoutCtx,"/api/feed", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to set feed mode: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- create_alert_rule ---

func createAlertRuleTool() mcp.Tool {
	return mcp.NewTool("create_alert_rule",
		mcp.WithDescription("Create an alert threshold rule for a probe. The rule fires when the condition is met and sends a notification. Use get_current_parameters to find valid probe names. Conditions: 'above' requires threshold_high; 'below' requires threshold_low; 'outside_range' requires both."),
		mcp.WithString("probe_name",
			mcp.Required(),
			mcp.Description("Probe name exactly as returned by get_current_parameters"),
		),
		mcp.WithString("condition",
			mcp.Required(),
			mcp.Description("Trigger condition: above/below fires when value crosses a threshold; outside_range fires when value leaves the band"),
			mcp.Enum("above", "below", "outside_range"),
		),
		mcp.WithNumber("threshold_low",
			mcp.Description("Low threshold — required for 'below' and 'outside_range' conditions"),
		),
		mcp.WithNumber("threshold_high",
			mcp.Description("High threshold — required for 'above' and 'outside_range' conditions"),
		),
		mcp.WithString("severity",
			mcp.Description("Notification severity (default: warning)"),
			mcp.Enum("warning", "critical"),
		),
		mcp.WithNumber("cooldown_minutes",
			mcp.Description("Minimum minutes between repeat notifications for the same rule (default: 15)"),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Whether the rule is active immediately (default: true)"),
		),
	)
}

func createAlertRuleHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		probeName, err := request.RequireString("probe_name")
		if err != nil {
			return toolError("Parameter 'probe_name' is required"), nil
		}
		condition, err := request.RequireString("condition")
		if err != nil {
			return toolError("Parameter 'condition' is required"), nil
		}

		body := map[string]any{
			"probe_name": probeName,
			"condition":  condition,
		}
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if v, ok := args["threshold_low"]; ok {
				body["threshold_low"] = v
			}
			if v, ok := args["threshold_high"]; ok {
				body["threshold_high"] = v
			}
			if v, ok := args["enabled"]; ok {
				if b, ok := v.(bool); ok {
					body["enabled"] = b
				}
			}
		}
		if v := request.GetString("severity", ""); v != "" {
			body["severity"] = v
		}
		if v := request.GetInt("cooldown_minutes", 0); v > 0 {
			body["cooldown_minutes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx,"/api/alerts", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to create alert rule: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- update_alert_rule ---

func updateAlertRuleTool() mcp.Tool {
	return mcp.NewTool("update_alert_rule",
		mcp.WithDescription("Update an existing alert rule. Only provide the fields you want to change — other fields are preserved. Use get_alert_rules to find the rule ID."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Alert rule ID from get_alert_rules"),
		),
		mcp.WithString("probe_name",
			mcp.Description("Probe name"),
		),
		mcp.WithString("condition",
			mcp.Description("Trigger condition"),
			mcp.Enum("above", "below", "outside_range"),
		),
		mcp.WithNumber("threshold_low",
			mcp.Description("Low threshold"),
		),
		mcp.WithNumber("threshold_high",
			mcp.Description("High threshold"),
		),
		mcp.WithString("severity",
			mcp.Description("Notification severity"),
			mcp.Enum("warning", "critical"),
		),
		mcp.WithNumber("cooldown_minutes",
			mcp.Description("Minimum minutes between repeat notifications"),
		),
		mcp.WithBoolean("enabled",
			mcp.Description("Whether the rule is active"),
		),
	)
}

func updateAlertRuleHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		targetID := int64(idFloat)
		idStr := fmt.Sprintf("%d", targetID)

		// Fetch the full list and find the rule — no single-item GET endpoint exists.
		var listResp struct {
			Rules []struct {
				ID              int64    `json:"id"`
				ProbeName       string   `json:"probe_name"`
				Condition       string   `json:"condition"`
				ThresholdLow    *float64 `json:"threshold_low"`
				ThresholdHigh   *float64 `json:"threshold_high"`
				Severity        string   `json:"severity"`
				CooldownMinutes int      `json:"cooldown_minutes"`
				Enabled         bool     `json:"enabled"`
			} `json:"rules"`
		}
		if err := apiCall(clientFromCtx(ctx, client),"/api/alerts", &listResp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}

		var existing *struct {
			ID              int64    `json:"id"`
			ProbeName       string   `json:"probe_name"`
			Condition       string   `json:"condition"`
			ThresholdLow    *float64 `json:"threshold_low"`
			ThresholdHigh   *float64 `json:"threshold_high"`
			Severity        string   `json:"severity"`
			CooldownMinutes int      `json:"cooldown_minutes"`
			Enabled         bool     `json:"enabled"`
		}
		for i := range listResp.Rules {
			if listResp.Rules[i].ID == targetID {
				existing = &listResp.Rules[i]
				break
			}
		}
		if existing == nil {
			return toolError(fmt.Sprintf("Alert rule %s not found", idStr)), nil
		}

		// Start with existing values.
		body := map[string]any{
			"probe_name":       existing.ProbeName,
			"condition":        existing.Condition,
			"severity":         existing.Severity,
			"cooldown_minutes": existing.CooldownMinutes,
			"enabled":          existing.Enabled,
		}
		if existing.ThresholdLow != nil {
			body["threshold_low"] = *existing.ThresholdLow
		}
		if existing.ThresholdHigh != nil {
			body["threshold_high"] = *existing.ThresholdHigh
		}

		// Apply overrides.
		if v := request.GetString("probe_name", ""); v != "" {
			body["probe_name"] = v
		}
		if v := request.GetString("condition", ""); v != "" {
			body["condition"] = v
		}
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if v, ok := args["threshold_low"]; ok {
				body["threshold_low"] = v
			}
			if v, ok := args["threshold_high"]; ok {
				body["threshold_high"] = v
			}
			if v, ok := args["enabled"]; ok {
				if b, ok := v.(bool); ok {
					body["enabled"] = b
				}
			}
		}
		if v := request.GetString("severity", ""); v != "" {
			body["severity"] = v
		}
		if v := request.GetInt("cooldown_minutes", 0); v > 0 {
			body["cooldown_minutes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Put(timeoutCtx,"/api/alerts/"+idStr, body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to update alert rule: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- delete_alert_rule ---

func deleteAlertRuleTool() mcp.Tool {
	return mcp.NewTool("delete_alert_rule",
		mcp.WithDescription("Delete an alert rule permanently. Use get_alert_rules to find the rule ID."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Alert rule ID from get_alert_rules"),
		),
	)
}

func deleteAlertRuleHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := clientFromCtx(ctx, client).Delete(timeoutCtx,"/api/alerts/"+idStr); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Alert rule %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to delete alert rule: %v", err)), nil
		}
		return mcp.NewToolResultText("Alert rule " + idStr + " deleted"), nil
	}
}

// --- list_probe_configs ---

func listProbeConfigsTool() mcp.Tool {
	return mcp.NewTool("list_probe_configs",
		mcp.WithDescription("List the display-band configuration for all probes: min/max normal (healthy range shown in UI) and min/max warning (outer limits before gauge turns red). Use before update_probe_config to see current values."),
	)
}

func listProbeConfigsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Get(timeoutCtx,"/api/config/probes", &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to list probe configs: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- update_probe_config ---

func updateProbeConfigTool() mcp.Tool {
	return mcp.NewTool("update_probe_config",
		mcp.WithDescription("Set the display-band configuration for a probe. min_normal/max_normal define the healthy range shown in the dashboard gauge; min_warning/max_warning are the outer limits before the gauge turns red. Use list_probe_configs + get_probe_history to calibrate values before calling this."),
		mcp.WithString("probe_name",
			mcp.Required(),
			mcp.Description("Probe name exactly as returned by get_current_parameters"),
		),
		mcp.WithNumber("min_normal",
			mcp.Description("Low end of the normal (healthy) range"),
		),
		mcp.WithNumber("max_normal",
			mcp.Description("High end of the normal (healthy) range"),
		),
		mcp.WithNumber("min_warning",
			mcp.Description("Low end of the warning range — below this the gauge turns red"),
		),
		mcp.WithNumber("max_warning",
			mcp.Description("High end of the warning range — above this the gauge turns red"),
		),
	)
}

func updateProbeConfigHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		probeName, err := request.RequireString("probe_name")
		if err != nil {
			return toolError("Parameter 'probe_name' is required"), nil
		}

		body := map[string]any{}
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			for _, key := range []string{"min_normal", "max_normal", "min_warning", "max_warning"} {
				if v, ok := args[key]; ok {
					body[key] = v
				}
			}
		}
		if len(body) == 0 {
			return toolError("At least one of min_normal, max_normal, min_warning, max_warning is required"), nil
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Put(timeoutCtx,"/api/config/probes/"+url.PathEscape(probeName), body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to update probe config for %s: %v", probeName, err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_journal_templates ---

func getJournalTemplatesTool() mcp.Tool {
	return mcp.NewTool("get_journal_templates",
		mcp.WithDescription("Get available journal entry templates grouped by category (observation, maintenance, event, milestone). Templates provide suggested titles and formats for common tank events — use them to create consistent, well-formed journal entries."),
	)
}

func getJournalTemplatesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/journal/templates", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_agent_context ---

func getAgentContextTool() mcp.Tool {
	return mcp.NewTool("get_agent_context",
		mcp.WithDescription("Get the full AI assistant context for this Symbiont tank: persona settings, tank profile (volume, dimensions, type), active livestock inventory, target parameter ranges derived from alert rules, dosing product preferences, custom guardrails, and a list of installed skills. Call this at the start of every skill workflow to load tank-specific facts before making recommendations."),
	)
}

func getAgentContextHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp struct {
			Context string `json:"context"`
		}
		if err := apiCall(clientFromCtx(ctx, client),"/api/agent/context", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return mcp.NewToolResultText(resp.Context), nil
	}
}

// --- list_skills ---

func listSkillsTool() mcp.Tool {
	return mcp.NewTool("list_skills",
		mcp.WithDescription("List all available Symbiont skill workflows with their names, descriptions, and enabled status. Use this to show the user which skills are available and whether they are currently enabled for install."),
	)
}

func listSkillsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client),"/api/agent/skills", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_skill ---

func getSkillTool() mcp.Tool {
	return mcp.NewTool("get_skill",
		mcp.WithDescription("Get the full content of a named Symbiont skill file, including the frontmatter and workflow instructions. Use this to inspect a skill before installing it."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The skill name (e.g. water-test-analysis, weekly-maintenance)."),
		),
	)
}

func getSkillHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError("name is required"), nil
		}
		var resp struct {
			Name string `json:"name"`
			Body string `json:"body"`
		}
		if err := apiCall(clientFromCtx(ctx, client),"/api/agent/skills/"+url.PathEscape(name)+"/body", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return mcp.NewToolResultText(resp.Body), nil
	}
}
