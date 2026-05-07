package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_due_tasks ---

func getDueTasksTool() mcp.Tool {
	return mcp.NewTool("get_due_tasks",
		mcp.WithDescription("Get all dosing and maintenance tasks that are currently due or overdue. Use to tell the user what care actions are needed today and what is overdue."),
	)
}

func getDueTasksHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/tasks/due", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- list_maintenance_tasks ---

func listMaintenanceTasksTool() mcp.Tool {
	return mcp.NewTool("list_maintenance_tasks",
		mcp.WithDescription("Get all maintenance tasks (water changes, glass cleaning, skimmer service, etc.) with their frequency, last completed time, and next due time. Use to understand the full maintenance schedule, not just what is due today."),
	)
}

func listMaintenanceTasksHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/maintenance/tasks", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- complete_maintenance_task ---

func completeMaintenanceTaskTool() mcp.Tool {
	return mcp.NewTool("complete_maintenance_task",
		mcp.WithDescription("Mark a maintenance task as completed. Use when the user says they did a water change, cleaned the glass, serviced the skimmer, or completed any other scheduled maintenance task."),
		mcp.WithNumber("task_id",
			mcp.Required(),
			mcp.Description("ID of the maintenance task (from get_due_tasks or list of tasks)."),
		),
		mcp.WithString("completed_at",
			mcp.Description("ISO 8601 timestamp of when the task was completed. Defaults to now."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes about this completion — e.g. 'changed 15 gallons'."),
		),
	)
}

func completeMaintenanceTaskHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireInt("task_id")
		if err != nil {
			return toolError("task_id is required"), nil
		}

		body := map[string]any{"source": "ai"}
		if v := request.GetString("completed_at", ""); v != "" {
			body["completed_at"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		path := fmt.Sprintf("/api/maintenance/tasks/%d/complete", taskID)
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, path, body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to complete task: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- create_maintenance_task ---

func createMaintenanceTaskTool() mcp.Tool {
	return mcp.NewTool("create_maintenance_task",
		mcp.WithDescription("Create a new recurring maintenance task (e.g. water change, glass cleaning, GFO reactor change, carbon swap). Use when the user wants to add a task to their maintenance schedule."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Short name for the task, e.g. 'Water Change', 'GFO Reactor Change'."),
		),
		mcp.WithString("frequency",
			mcp.Required(),
			mcp.Enum("daily", "every_n_days", "weekly", "monthly", "as_needed"),
			mcp.Description("How often the task recurs. Use every_n_days with interval_days for custom intervals like 'every 2 weeks'."),
		),
		mcp.WithString("description",
			mcp.Description("Optional longer description or instructions for the task."),
		),
		mcp.WithNumber("interval_days",
			mcp.Description("Number of days between occurrences. Required when frequency is every_n_days (e.g. 14 for biweekly)."),
		),
		mcp.WithNumber("day_of_week",
			mcp.Description("Day of week for weekly tasks: 0=Sunday, 1=Monday … 6=Saturday."),
		),
	)
}

func createMaintenanceTaskHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError("name is required"), nil
		}
		frequency, err := request.RequireString("frequency")
		if err != nil {
			return toolError("frequency is required"), nil
		}

		body := map[string]any{
			"name":      name,
			"frequency": frequency,
		}
		if v := request.GetString("description", ""); v != "" {
			body["description"] = v
		}
		if v := request.GetFloat("interval_days", -1); v > 0 {
			body["interval_days"] = v
		}
		if v := request.GetFloat("day_of_week", -1); v >= 0 {
			body["day_of_week"] = int(v)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/maintenance/tasks", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to create task: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
