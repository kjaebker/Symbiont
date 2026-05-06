package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_system_status ---

func getSystemStatusTool() mcp.Tool {
	return mcp.NewTool("get_system_status",
		mcp.WithDescription("Get Apex controller information (firmware, serial number) and Symbiont system health (last poll time, whether polling is working, database sizes). Use to confirm the system is functioning normally."),
	)
}

func getSystemStatusHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/system", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- summarize_tank_health ---

func summarizeTankHealthTool() mcp.Tool {
	return mcp.NewTool("summarize_tank_health",
		mcp.WithDescription("Get a comprehensive health snapshot of the aquarium — all current parameters with status, outlet states, and system health. Best starting point for a general tank status check."),
	)
}

func summarizeTankHealthHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		type probeEntry struct {
			Name   string  `json:"name"`
			Value  float64 `json:"value"`
			Unit   string  `json:"unit"`
			Status string  `json:"status"`
		}
		type outletEntry struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
			Type  string `json:"type"`
		}

		type livestockEntry struct {
			ID       int64   `json:"id"`
			Name     string  `json:"name"`
			Type     string  `json:"type"`
			Quantity int     `json:"quantity"`
			Status   string  `json:"status"`
			Species  *string `json:"species"`
		}

		var (
			probesResp struct {
				Probes []probeEntry `json:"probes"`
			}
			outletsResp struct {
				Outlets []outletEntry `json:"outlets"`
			}
			systemResp struct {
				PollOK   bool   `json:"poll_ok"`
				LastPoll string `json:"last_poll"`
			}
			livestockResp struct {
				Livestock []livestockEntry `json:"livestock"`
			}
			errs [4]error
			wg   sync.WaitGroup
		)

		wg.Add(4)
		go func() {
			defer wg.Done()
			errs[0] = apiCall(clientFromCtx(ctx, client), "/api/probes", &probesResp)
		}()
		go func() {
			defer wg.Done()
			errs[1] = apiCall(clientFromCtx(ctx, client), "/api/outlets", &outletsResp)
		}()
		go func() {
			defer wg.Done()
			errs[2] = apiCall(clientFromCtx(ctx, client), "/api/system", &systemResp)
		}()
		go func() {
			defer wg.Done()
			errs[3] = apiCall(clientFromCtx(ctx, client), "/api/livestock", &livestockResp)
		}()
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
			}
		}

		// Synthesize health summary.
		var warnings, critical []string
		allNormal := true
		for _, p := range probesResp.Probes {
			switch p.Status {
			case "warning":
				warnings = append(warnings, p.Name)
				allNormal = false
			case "critical":
				critical = append(critical, p.Name)
				allNormal = false
			}
		}

		var onCount, offCount, autoCount int
		for _, o := range outletsResp.Outlets {
			switch strings.ToUpper(o.State) {
			case "ON", "AON":
				onCount++
			case "OFF", "AOF":
				offCount++
			default:
				autoCount++
			}
		}

		// Aggregate livestock by type (sum quantities) and collect non-healthy items.
		byType := map[string]int{}
		type nonHealthyEntry struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Status string `json:"status"`
		}
		var nonHealthy []nonHealthyEntry
		for _, item := range livestockResp.Livestock {
			byType[item.Type] += item.Quantity
			if item.Status != "healthy" {
				nonHealthy = append(nonHealthy, nonHealthyEntry{
					ID:     item.ID,
					Name:   item.Name,
					Type:   item.Type,
					Status: item.Status,
				})
			}
		}
		if nonHealthy == nil {
			nonHealthy = []nonHealthyEntry{}
		}

		totalLivestock := 0
		for _, q := range byType {
			totalLivestock += q
		}

		summary := map[string]any{
			"system_ok":    systemResp.PollOK,
			"poll_ok":      systemResp.PollOK,
			"last_poll_ts": systemResp.LastPoll,
			"parameters": map[string]any{
				"all_normal": allNormal,
				"warnings":   warnings,
				"critical":   critical,
				"probes":     probesResp.Probes,
			},
			"outlets": map[string]any{
				"total":   len(outletsResp.Outlets),
				"on":      onCount,
				"off":     offCount,
				"auto":    autoCount,
				"outlets": outletsResp.Outlets,
			},
			"livestock": map[string]any{
				"total":       totalLivestock,
				"by_type":     byType,
				"non_healthy": nonHealthy,
			},
		}

		return jsonResult(summary)
	}
}

// --- get_system_log ---

func getSystemLogTool() mcp.Tool {
	return mcp.NewTool("get_system_log",
		mcp.WithDescription("Get recent structured log entries from the Symbiont API and poller services. Useful for diagnosing errors, checking poll failures, or understanding recent system activity."),
		mcp.WithNumber("limit",
			mcp.Description("Max log lines to return, default 50, max 500"),
		),
		mcp.WithString("service",
			mcp.Description("Filter by service: api or poller. Omit for both."),
			mcp.Enum("api", "poller"),
		),
	)
}

func getSystemLogHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", request.GetInt("limit", 50)))
		if v := request.GetString("service", ""); v != "" {
			q.Set("service", v)
		}
		path := "/api/system/log?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_devices ---

func getDevicesTool() mcp.Tool {
	return mcp.NewTool("get_devices",
		mcp.WithDescription("Get the list of aquarium devices (equipment) configured in Symbiont — pumps, lights, skimmers, heaters, etc. Each device may have an associated outlet and probes. Useful for understanding what equipment is in the tank and how it relates to outlets and sensor readings."),
	)
}

func getDevicesHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/devices", &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}
