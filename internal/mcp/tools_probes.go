package mcp

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_current_parameters ---

func getCurrentParametersTool() mcp.Tool {
	return mcp.NewTool("get_current_parameters",
		mcp.WithDescription("Get the current reading for all aquarium probes — temperature, pH, ORP, salinity, and any others connected to the Apex. Returns current value, unit, status (normal/warning/critical), and timestamp of last reading."),
	)
}

func getCurrentParametersHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/probes", &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Probe '%s' not found or no data in range", name)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
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
		if err := clientFromCtx(ctx, client).Get(timeoutCtx, "/api/config/probes", &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Put(timeoutCtx, "/api/config/probes/"+url.PathEscape(probeName), body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to update probe config for %s: %v", probeName, err)), nil
		}
		return jsonResult(resp)
	}
}
