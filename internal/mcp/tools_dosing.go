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

// --- get_dosing_schedule ---

// --- get_dosing_products ---

func getDosingProductsTool() mcp.Tool {
	return mcp.NewTool("get_dosing_products",
		mcp.WithDescription("List all dosing products (supplements, filter media, etc.) registered in Symbiont, with their IDs, brand, name, type, and unit. Call this before create_dosing_schedule to get the product_id when the product doesn't already have a schedule."),
	)
}

func getDosingProductsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/dosing/products", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- create_dosing_schedule ---

func createDosingScheduleTool() mcp.Tool {
	return mcp.NewTool("create_dosing_schedule",
		mcp.WithDescription("Create a new dosing schedule for a product. Use when the user wants to start dosing a supplement or schedule a recurring product addition. Call get_dosing_products first if you don't already know the product_id."),
		mcp.WithNumber("product_id",
			mcp.Required(),
			mcp.Description("ID of the product to schedule (from get_dosing_products or get_dosing_schedule)."),
		),
		mcp.WithNumber("amount",
			mcp.Required(),
			mcp.Description("Amount to dose per occurrence, in the product's unit (e.g. 5 for 5 mL)."),
		),
		mcp.WithString("frequency",
			mcp.Required(),
			mcp.Enum("daily", "twice_daily", "every_n_days", "weekly", "as_needed"),
			mcp.Description("How often to dose. Use every_n_days with interval_days for custom intervals."),
		),
		mcp.WithNumber("interval_days",
			mcp.Description("Days between doses. Required when frequency is every_n_days (e.g. 3 for every 3 days)."),
		),
		mcp.WithNumber("day_of_week",
			mcp.Description("Day of week for weekly doses: 0=Sunday, 1=Monday … 6=Saturday."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes about this schedule, e.g. 'increase if alk drops below 8'."),
		),
	)
}

func createDosingScheduleHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		productID := request.GetFloat("product_id", 0)
		if productID == 0 {
			return toolError("product_id is required"), nil
		}
		amount := request.GetFloat("amount", 0)
		if amount <= 0 {
			return toolError("amount must be greater than 0"), nil
		}
		frequency, err := request.RequireString("frequency")
		if err != nil {
			return toolError("frequency is required"), nil
		}

		body := map[string]any{
			"product_id": int64(productID),
			"amount":     amount,
			"frequency":  frequency,
		}
		if v := request.GetFloat("interval_days", -1); v > 0 {
			body["interval_days"] = v
		}
		if v := request.GetFloat("day_of_week", -1); v >= 0 {
			body["day_of_week"] = int(v)
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/dosing/schedules", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to create dosing schedule: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func getDosingScheduleTool() mcp.Tool {
	return mcp.NewTool("get_dosing_schedule",
		mcp.WithDescription("Get the current dosing schedule — all enabled products with their dose amounts, frequency, last dosed time, and next due time. Use to understand what supplements the tank receives and when."),
	)
}

func getDosingScheduleHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/dosing/schedules", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- log_dose ---

func logDoseTool() mcp.Tool {
	return mcp.NewTool("log_dose",
		mcp.WithDescription("Log that a dose was administered. Use when the user says they dosed a supplement or asks to record a dose. Provide the schedule_id from get_dosing_schedule to record against a scheduled dose, or provide product_id and amount for an ad-hoc dose."),
		mcp.WithNumber("schedule_id",
			mcp.Description("ID of the dosing schedule (from get_dosing_schedule). Preferred when logging a scheduled dose — automatically advances next_due_at."),
		),
		mcp.WithNumber("product_id",
			mcp.Description("Product ID for an ad-hoc dose (not tied to a schedule). Required if schedule_id is not provided."),
		),
		mcp.WithNumber("amount",
			mcp.Description("Amount dosed. If schedule_id is provided and amount is omitted, uses the scheduled amount."),
		),
		mcp.WithString("dosed_at",
			mcp.Description("ISO 8601 timestamp of when the dose was given. Defaults to now."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes about this dose."),
		),
	)
}

func logDoseHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scheduleID := request.GetFloat("schedule_id", 0)
		productID := request.GetFloat("product_id", 0)

		if scheduleID == 0 && productID == 0 {
			return toolError("Either schedule_id or product_id is required"), nil
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if scheduleID > 0 {
			// Scheduled dose — POST to /api/dosing/schedules/{id}/log
			body := map[string]any{"source": "ai"}
			if v := request.GetFloat("amount", 0); v > 0 {
				body["amount"] = v
			}
			if v := request.GetString("dosed_at", ""); v != "" {
				body["dosed_at"] = v
			}
			if v := request.GetString("notes", ""); v != "" {
				body["notes"] = v
			}
			var resp any
			path := fmt.Sprintf("/api/dosing/schedules/%d/log", int64(scheduleID))
			if err := clientFromCtx(ctx, client).Post(timeoutCtx, path, body, &resp); err != nil {
				return toolError(fmt.Sprintf("Failed to log dose: %v", err)), nil
			}
			return jsonResult(resp)
		}

		// Ad-hoc dose — not yet exposed as a separate endpoint; use schedule log with product info
		return toolError("Ad-hoc doses (without schedule_id) are not yet supported. Use get_dosing_schedule to find the schedule_id."), nil
	}
}

// --- get_dosing_history ---

func getDosingHistoryTool() mcp.Tool {
	return mcp.NewTool("get_dosing_history",
		mcp.WithDescription("Get recent dosing history — what was dosed, when, how much. Useful for reviewing supplement consistency and correlating dosing with water chemistry changes."),
		mcp.WithNumber("product_id",
			mcp.Description("Filter to a specific product ID. Omit for all products."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of log entries to return. Defaults to 50, max 100."),
		),
	)
}

func getDosingHistoryHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := request.GetFloat("product_id", 0); v > 0 {
			q.Set("product_id", fmt.Sprintf("%d", int64(v)))
		}
		limit := min(request.GetInt("limit", 50), 100)
		q.Set("limit", fmt.Sprintf("%d", limit))

		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/dosing/logs?"+q.Encode(), &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
