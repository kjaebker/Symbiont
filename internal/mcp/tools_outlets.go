package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_outlet_states ---

func getOutletStatesTool() mcp.Tool {
	return mcp.NewTool("get_outlet_states",
		mcp.WithDescription("Get the current state of all outlets controlled by the Apex. State values: ON/OFF (manual override), AON/AOF (auto mode, program running). Includes outlet type and intensity."),
	)
}

func getOutletStatesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/outlets", &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Put(ctx, "/api/outlets/"+id, map[string]string{"state": state}, &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
