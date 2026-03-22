# Symbiont — Phase 5: MCP Server
> AI integration via Model Context Protocol

**Deliverable:** Claude can query tank parameters, view outlet states, control outlets, and get a health summary through MCP. Tested against Claude Desktop and Claude Code.

---

## 5.1 MCP Dependency and Server Setup

↳ depends on: Phase 3 (CLI) complete — confirms API client patterns are solid

- [x] [code] Add dependency: `go get github.com/mark3labs/mcp-go`
- [x] [code] Create `internal/mcp/tools.go`:
  - [x] `RegisterTools(s *server.MCPServer, client *cli.APIClient)` — registers all tools
  - [x] Reuses `internal/cli.APIClient` for API calls (shared client pattern)
- [x] [code] Create `cmd/mcp/main.go`:
  - [x] Load config (needs `SYMBIONT_API_URL` and `SYMBIONT_TOKEN`)
  - [x] Create API client
  - [x] Create MCP server with tools registered
  - [x] Call `server.ServeStdio()` — blocks on stdin/stdout
  - [x] Log startup to `stderr` (not stdout — stdout is MCP protocol)
- [x] [verify] `go build ./cmd/mcp` compiles

---

## 5.2 Tool Implementations

↳ depends on: 5.1

### get_current_parameters

- [x] [code] Register tool — returns all probe readings with value, unit, status, timestamp
- [x] [test] Returns correct probe data

### get_probe_history

- [x] [code] Register tool with name, from, to, interval params
- [x] [test] Returns bucketed data for valid probe
- [x] [test] Returns clear error for unknown probe (404)

### get_outlet_states

- [x] [code] Register tool — returns all outlet states
- [x] [test] Returns outlet data

### control_outlet

- [x] [code] Register tool with id and state (ON/OFF) params (AUTO not supported by Apex REST API)
- [x] [code] Validates state, calls PUT /api/outlets/<id>
- [x] [test] Successfully sets outlet state
- [x] [test] Returns error for invalid state

### get_outlet_event_log

- [x] [code] Register tool with optional outlet_id and limit params
- [x] [test] Returns event data

### get_alert_rules

- [x] [code] Register tool — returns all alert rules
- [x] [test] Returns alert data

### get_system_status

- [x] [code] Register tool — returns controller info and system health
- [x] [test] Returns system data with serial number

### summarize_tank_health

- [x] [code] Composite tool — concurrent calls to probes, outlets, system
- [x] [code] Synthesizes into health summary with all_normal, warnings, critical arrays
- [x] [test] Returns complete health snapshot
- [x] [test] Correctly identifies probes in warning state

---

## 5.3 Error Handling in Tools

- [x] [code] All tool handlers return structured errors via IsError flag
- [x] [code] API unreachable → clear "Cannot reach Symbiont API" message
- [x] [code] Invalid input → descriptive error message
- [x] [code] Unknown probe → "not found" error
- [x] [code] All API calls wrapped with 10-second timeout context
- [x] [test] Tool with API down returns clear error message

---

## 5.4 Claude Desktop Integration

- [ ] [config] Build `symbiont-mcp` binary and place at a stable path
- [ ] [config] Create Claude Desktop MCP config
- [ ] [verify] Claude Desktop connects to MCP server without error
- [ ] [verify] Claude can list available tools
- [ ] [verify] Claude correctly calls `get_current_parameters`
- [ ] [verify] Claude correctly calls `summarize_tank_health`
- [ ] [verify] Claude correctly calls `control_outlet` after confirming intent

---

## 5.5 Claude Code Integration

- [ ] [config] Add MCP server to Claude Code config
- [ ] [verify] Claude Code has `symbiont` MCP server available
- [ ] [verify] "Check my tank parameters" returns data
- [ ] [verify] Natural language outlet control works

---

## 5.6 NixOS MCP Service

- [ ] [config] Evaluate whether systemd service is needed (MCP stdio servers are typically launched as subprocesses)
- [ ] [decision] Defer SSE transport decision to after initial stdio testing

---

## 5.7 Integration Testing

- [x] [test] Create `internal/mcp/tools_test.go`:
  - [x] Mock API server with `httptest.NewServer`
  - [x] Seed test probe and outlet data
  - [x] Test each tool handler returns correct JSON structure
  - [x] Test `summarize_tank_health` makes concurrent API calls
  - [x] Test error handling for API failures
- [x] [verify] `go test ./internal/mcp/...` passes (11 tests)

---

## 5.8 Interaction Testing (Manual Scenarios)

- [ ] [verify] "What's my tank temperature right now?"
- [ ] [verify] "Has my pH been stable in the last 24 hours?"
- [ ] [verify] "Is everything in my tank normal?"
- [ ] [verify] "Turn off my skimmer"
- [ ] [verify] "What changed in my tank yesterday afternoon?"

---

## Phase 5 Checklist Summary

- [x] `mcp-go` dependency integrated
- [x] All 8 tools implemented and tested
- [x] Error handling returns clear messages
- [ ] Claude Desktop integration verified
- [ ] Claude Code integration verified
- [ ] All manual interaction scenarios pass

**Phase 5 is complete when:** Claude Desktop can answer "Is my tank healthy?" with real data from Symbiont, and "Turn off my return pump" physically toggles the outlet.
