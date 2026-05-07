package mcp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
		if err := apiCall(clientFromCtx(ctx, client), "/api/agent/context", &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/agent/skills", &resp); err != nil {
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/agent/skills/"+url.PathEscape(name)+"/body", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return mcp.NewToolResultText(resp.Body), nil
	}
}
