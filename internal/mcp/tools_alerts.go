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

// --- get_alert_rules ---

func getAlertRulesTool() mcp.Tool {
	return mcp.NewTool("get_alert_rules",
		mcp.WithDescription("Get all configured alert rules — the thresholds set for each probe that trigger notifications when breached. Useful for understanding what parameter ranges are considered normal or concerning."),
	)
}

func getAlertRulesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/alerts", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
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
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
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
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/alerts", body, &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/alerts", &listResp); err != nil {
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
		if err := clientFromCtx(ctx, client).Put(timeoutCtx, "/api/alerts/"+idStr, body, &resp); err != nil {
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

		if err := clientFromCtx(ctx, client).Delete(timeoutCtx, "/api/alerts/"+idStr); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Alert rule %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to delete alert rule: %v", err)), nil
		}
		return mcp.NewToolResultText("Alert rule " + idStr + " deleted"), nil
	}
}
