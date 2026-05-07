package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_feed_mode ---

func getFeedModeTool() mcp.Tool {
	return mcp.NewTool("get_feed_mode",
		mcp.WithDescription("Get the current feed mode status. Feed modes temporarily suspend certain equipment (e.g. powerheads) to allow feeding without livestock getting caught. Name values: 0=off, 1=Feed A, 2=Feed B, 3=Feed C, 4=Feed D."),
	)
}

func getFeedModeHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/feed", &resp); err != nil {
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
		if err := clientFromCtx(ctx, client).Put(timeoutCtx, "/api/feed", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to set feed mode: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
