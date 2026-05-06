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
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/measurements", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add measurement: %v", err)), nil
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/measurements/parameters", &resp); err != nil {
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

		if err := clientFromCtx(ctx, client).Delete(timeoutCtx, "/api/measurements/"+idStr); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Measurement %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to delete measurement: %v", err)), nil
		}
		return mcp.NewToolResultText("Measurement " + idStr + " deleted"), nil
	}
}
