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
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/journal", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add journal entry: %v", err)), nil
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/journal/templates", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
