package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools adds all Symbiont tools to the MCP server.
func RegisterTools(s *server.MCPServer, client *cli.APIClient) {
	s.AddTool(getCurrentParametersTool(), getCurrentParametersHandler(client))
	s.AddTool(getProbeHistoryTool(), getProbeHistoryHandler(client))
	s.AddTool(getOutletStatesTool(), getOutletStatesHandler(client))
	s.AddTool(controlOutletTool(), controlOutletHandler(client))
	s.AddTool(getOutletEventLogTool(), getOutletEventLogHandler(client))
	s.AddTool(getAlertRulesTool(), getAlertRulesHandler(client))
	s.AddTool(getAlertEventsTool(), getAlertEventsHandler(client))
	s.AddTool(createAlertRuleTool(), createAlertRuleHandler(client))
	s.AddTool(updateAlertRuleTool(), updateAlertRuleHandler(client))
	s.AddTool(deleteAlertRuleTool(), deleteAlertRuleHandler(client))
	s.AddTool(getSystemStatusTool(), getSystemStatusHandler(client))
	s.AddTool(getSystemLogTool(), getSystemLogHandler(client))
	s.AddTool(getDevicesTool(), getDevicesHandler(client))
	s.AddTool(summarizeTankHealthTool(), summarizeTankHealthHandler(client))
	s.AddTool(getMeasurementParametersTool(), getMeasurementParametersHandler(client))
	s.AddTool(getMeasurementsTool(), getMeasurementsHandler(client))
	s.AddTool(addMeasurementTool(), addMeasurementHandler(client))
	s.AddTool(deleteMeasurementTool(), deleteMeasurementHandler(client))
	s.AddTool(getLivestockTool(), getLivestockHandler(client))
	s.AddTool(addLivestockTool(), addLivestockHandler(client))
	s.AddTool(updateLivestockTool(), updateLivestockHandler(client))
	s.AddTool(getLivestockObservationsTool(), getLivestockObservationsHandler(client))
	s.AddTool(addLivestockObservationTool(), addLivestockObservationHandler(client))
	s.AddTool(getLivestockImageTool(), getLivestockImageHandler(client))
	s.AddTool(getFeedModeTool(), getFeedModeHandler(client))
	s.AddTool(setFeedModeTool(), setFeedModeHandler(client))
	s.AddTool(getJournalEntriesTool(), getJournalEntriesHandler(client))
	s.AddTool(addJournalEntryTool(), addJournalEntryHandler(client))
	s.AddTool(getJournalTemplatesTool(), getJournalTemplatesHandler(client))
	s.AddTool(getTankProfileTool(), getTankProfileHandler(client))
	s.AddTool(getAgentContextTool(), getAgentContextHandler(client))
	s.AddTool(listSkillsTool(), listSkillsHandler(client))
	s.AddTool(getSkillTool(), getSkillHandler(client))
	s.AddTool(listProbeConfigsTool(), listProbeConfigsHandler(client))
	s.AddTool(updateProbeConfigTool(), updateProbeConfigHandler(client))
	s.AddTool(getDosingProductsTool(), getDosingProductsHandler(client))
	s.AddTool(getDosingScheduleTool(), getDosingScheduleHandler(client))
	s.AddTool(createDosingScheduleTool(), createDosingScheduleHandler(client))
	s.AddTool(logDoseTool(), logDoseHandler(client))
	s.AddTool(getDosingHistoryTool(), getDosingHistoryHandler(client))
	s.AddTool(getDueTasksTool(), getDueTasksHandler(client))
	s.AddTool(listMaintenanceTasksTool(), listMaintenanceTasksHandler(client))
	s.AddTool(completeMaintenanceTaskTool(), completeMaintenanceTaskHandler(client))
	s.AddTool(createMaintenanceTaskTool(), createMaintenanceTaskHandler(client))
}

type contextKeyClient struct{}

// WithLoopbackClient stores a per-request loopback API client in ctx.
// Called by the MCP HTTP handler so each tool invocation uses the caller's token.
func WithLoopbackClient(ctx context.Context, client *cli.APIClient) context.Context {
	return context.WithValue(ctx, contextKeyClient{}, client)
}

// clientFromCtx returns the per-request loopback client from ctx,
// falling back to the static client (used in stdio mode).
func clientFromCtx(ctx context.Context, fallback *cli.APIClient) *cli.APIClient {
	if c, ok := ctx.Value(contextKeyClient{}).(*cli.APIClient); ok {
		return c
	}
	return fallback
}

func apiCall(client *cli.APIClient, path string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Get(ctx, path, result)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(msg)},
		IsError: true,
	}
}
