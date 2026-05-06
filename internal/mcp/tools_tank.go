package mcp

import (
	"context"
	"fmt"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_tank_profile ---

func getTankProfileTool() mcp.Tool {
	return mcp.NewTool("get_tank_profile",
		mcp.WithDescription("Get the physical profile of the display tank and sump — volume, dimensions, tank type, manufacturer, model, and setup date. Useful for context when answering questions about stocking levels, equipment sizing, or how long the tank has been running."),
	)
}

func getTankProfileHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/tank/profile", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
