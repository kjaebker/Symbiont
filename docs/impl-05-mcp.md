# Symbiont — Phase 5: MCP Server
> AI integration via Model Context Protocol

**Deliverable:** Claude can query tank parameters, view outlet states, control outlets, and get a health summary through MCP. Tested against Claude Desktop and Claude Code.

---

## 5.1 MCP Dependency and Server Setup

↳ depends on: Phase 3 (CLI) complete — confirms API client patterns are solid

- [ ] [code] Add dependency: `go get github.com/mark3labs/mcp-go`
- [ ] [research] Review `mcp-go` docs for tool registration and stdio server patterns
- [ ] [code] Create `internal/mcp/server.go`:
  - [ ] `NewServer(apiClient *apiclient.Client) *mcp.Server`
  - [ ] Registers all tools (see 5.2)
  - [ ] Returns configured server ready to `ServeStdio()`
- [ ] [code] Create `cmd/mcp/main.go`:
  - [ ] Load config (needs `SYMBIONT_API_URL` and `SYMBIONT_TOKEN`)
  - [ ] Create API client
  - [ ] Create MCP server with tools registered
  - [ ] Call `server.ServeStdio()` — blocks on stdin/stdout
  - [ ] Log startup to `stderr` (not stdout — stdout is MCP protocol)
- [ ] [code] Create `internal/mcp/apiclient.go`:
  - [ ] Thin wrapper over `net/http` for MCP server to call the Symbiont API
  - [ ] Reuses same pattern as CLI client (`internal/cli/client.go`)
  - [ ] Can share the same client type — consider extracting to `internal/apiclient/`
- [ ] [verify] `go build ./cmd/mcp` compiles
- [ ] [verify] `echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./mcp` returns tool list

---

## 5.2 Tool Implementations

↳ depends on: 5.1

### get_current_parameters

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_current_parameters`
  - [ ] Description: "Get the current reading for all aquarium probes — temperature, pH, ORP, salinity, and any others connected to the Apex. Returns current value, unit, status (normal/warning/critical), and timestamp of last reading."
  - [ ] Input schema: none (empty object)
- [ ] [code] Handler:
  - [ ] Calls `GET /api/probes` via API client
  - [ ] Returns JSON response as tool result
- [ ] [verify] Tool returns correct data when called

### get_probe_history

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_probe_history`
  - [ ] Description: "Get time-series history for a specific probe. Useful for analyzing trends, correlating parameter changes with events, or understanding patterns over time."
  - [ ] Input schema:
    - [ ] `name` (string, required): probe name exactly as returned by get_current_parameters
    - [ ] `from` (string, optional): ISO 8601 start time, default 24 hours ago
    - [ ] `to` (string, optional): ISO 8601 end time, default now
    - [ ] `interval` (string, optional): bucket size — "10s", "1m", "5m", "15m", "1h", "1d" — default auto
- [ ] [code] Handler:
  - [ ] Calls `GET /api/probes/<name>/history` with params
  - [ ] Returns JSON response
- [ ] [verify] Tool returns bucketed data for a valid probe name
- [ ] [verify] Tool returns clear error for unknown probe name

### get_outlet_states

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_outlet_states`
  - [ ] Description: "Get the current state and power draw of all outlets controlled by the Apex. State can be ON, OFF, or AUTO (where xstatus shows the resolved physical state). Includes watts and amps per outlet."
  - [ ] Input schema: none
- [ ] [code] Handler:
  - [ ] Calls `GET /api/outlets`
  - [ ] Returns JSON response
- [ ] [verify] Tool returns real outlet states

### control_outlet

- [ ] [code] Register tool with schema:
  - [ ] Name: `control_outlet`
  - [ ] Description: "Set an outlet to ON, OFF, or AUTO. Use the outlet ID from get_outlet_states. AUTO returns the outlet to Apex program control. Use with care — this directly controls aquarium equipment."
  - [ ] Input schema:
    - [ ] `id` (string, required): outlet ID from get_outlet_states
    - [ ] `state` (string, required, enum: ["ON", "OFF", "AUTO"]): desired state
- [ ] [code] Handler:
  - [ ] Validates `state` is one of ON/OFF/AUTO
  - [ ] Calls `PUT /api/outlets/<id>` with `initiated_by` set to `"mcp"` (API server already does this)
  - [ ] Returns updated outlet state or error message
- [ ] [verify] Tool physically toggles an outlet via Claude
- [ ] [verify] Event log shows `initiated_by: mcp` in SQLite

### get_outlet_event_log

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_outlet_event_log`
  - [ ] Description: "Get a log of recent outlet state changes, including who or what made each change (ui, cli, mcp, api) and what the state changed from and to. Useful for understanding what happened in the tank over time."
  - [ ] Input schema:
    - [ ] `outlet_id` (string, optional): filter to specific outlet. Omit for all outlets.
    - [ ] `limit` (integer, optional): max events to return, default 20, max 100
- [ ] [code] Handler:
  - [ ] Calls `GET /api/outlets/<id>/events` or `GET /api/outlets/events` (all)
  - [ ] Returns JSON response
- [ ] [verify] Returns recent events in correct shape

### get_alert_rules

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_alert_rules`
  - [ ] Description: "Get all configured alert rules — the thresholds set for each probe that trigger notifications when breached. Useful for understanding what parameter ranges are considered normal or concerning."
  - [ ] Input schema: none
- [ ] [code] Handler:
  - [ ] Calls `GET /api/alerts`
  - [ ] Returns JSON response
- [ ] [verify] Returns configured rules

### get_system_status

- [ ] [code] Register tool with schema:
  - [ ] Name: `get_system_status`
  - [ ] Description: "Get Apex controller information (firmware, serial number) and Symbiont system health (last poll time, whether polling is working, database sizes). Use to confirm the system is functioning normally."
  - [ ] Input schema: none
- [ ] [code] Handler:
  - [ ] Calls `GET /api/system`
  - [ ] Returns JSON response
- [ ] [verify] Returns system status with `poll_ok: true`

### summarize_tank_health

- [ ] [code] Register tool with schema:
  - [ ] Name: `summarize_tank_health`
  - [ ] Description: "Get a comprehensive health snapshot of the aquarium — all current parameters with status, outlet states, any active alerts, and system health. Best starting point for a general tank status check."
  - [ ] Input schema: none
- [ ] [code] Handler — composite tool, makes three API calls:
  - [ ] Calls `GET /api/probes` concurrently
  - [ ] Calls `GET /api/outlets` concurrently
  - [ ] Calls `GET /api/system` concurrently
  - [ ] Use `sync.WaitGroup` or `errgroup` for concurrent fetches
  - [ ] Synthesizes into a single structured response:
    ```json
    {
      "system_ok": true,
      "poll_ok": true,
      "last_poll_ts": "...",
      "parameters": {
        "all_normal": false,
        "warnings": ["ORP", "pH"],
        "critical": [],
        "probes": [...]
      },
      "outlets": {
        "total": 8,
        "on": 6,
        "off": 1,
        "auto": 1,
        "outlets": [...]
      }
    }
    ```
- [ ] [verify] Single tool call returns complete tank picture
- [ ] [verify] `warnings` array correctly identifies probes in warning state

---

## 5.3 Error Handling in Tools

- [ ] [code] All tool handlers must return structured errors, not panics:
  - [ ] API unreachable → return `{ "error": "Cannot reach Symbiont API", "details": "..." }`
  - [ ] Invalid input → return `{ "error": "Invalid outlet state. Must be ON, OFF, or AUTO" }`
  - [ ] Unknown probe → return `{ "error": "Probe 'X' not found", "available_probes": [...] }`
- [ ] [code] Wrap all API calls with 10-second timeout context
- [ ] [verify] Calling tool with invalid input returns clear error message (not stack trace)
- [ ] [verify] Calling tool when API is down returns clear error message

---

## 5.4 Claude Desktop Integration

- [ ] [config] Build `symbiont-mcp` binary and place at a stable path (NixOS: `/run/current-system/sw/bin/symbiont-mcp`)
- [ ] [config] Create Claude Desktop MCP config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, or equivalent):
  ```json
  {
    "mcpServers": {
      "symbiont": {
        "command": "/run/current-system/sw/bin/symbiont-mcp",
        "env": {
          "SYMBIONT_API_URL": "http://localhost:8420",
          "SYMBIONT_TOKEN": "your-token-here"
        }
      }
    }
  }
  ```
- [ ] [verify] Claude Desktop connects to MCP server without error
- [ ] [verify] Claude can list available tools: "What tools do you have for my aquarium?"
- [ ] [verify] Claude correctly calls `get_current_parameters`
- [ ] [verify] Claude correctly calls `summarize_tank_health`
- [ ] [verify] Claude correctly calls `control_outlet` after confirming intent

---

## 5.5 Claude Code Integration

- [ ] [config] Add MCP server to Claude Code config:
  ```bash
  claude mcp add symbiont /run/current-system/sw/bin/symbiont-mcp \
    --env SYMBIONT_API_URL=http://localhost:8420 \
    --env SYMBIONT_TOKEN=your-token
  ```
- [ ] [verify] `claude` CLI has `symbiont` MCP server available
- [ ] [verify] "Check my tank parameters" in a Claude Code session returns data
- [ ] [verify] Natural language outlet control works from terminal

---

## 5.6 NixOS MCP Service

- [ ] [config] Add `symbiont-mcp` systemd service to `flake.nix`
- [ ] [verify] Service starts and stays running (`ServeStdio` blocks waiting for client connection — service should be `Type=simple`)
- [ ] [!] Note: MCP server in stdio mode may not need to be a persistent service — Claude Desktop/Code launch it as a subprocess. Evaluate whether systemd service is needed or if `ExecStart` path in desktop config is sufficient.
- [ ] [decision] If running as a persistent network MCP server (SSE transport instead of stdio): implement HTTP/SSE transport and bind to a port. This enables remote AI clients over Tailscale. Defer this decision to after initial stdio testing.

---

## 5.7 Integration Testing

- [ ] [test] Create `internal/mcp/tools_test.go`:
  - [ ] Mock API server with `httptest.NewServer`
  - [ ] Seed test probe and outlet data
  - [ ] Test each tool handler returns correct JSON structure
  - [ ] Test `summarize_tank_health` makes all three concurrent API calls
  - [ ] Test error handling for API failures
- [ ] [verify] `go test ./internal/mcp/...` passes

---

## 5.8 Interaction Testing (Manual Scenarios)

Document these as accepted conversation patterns once working:

- [ ] [verify] "What's my tank temperature right now?" → calls `get_current_parameters`, returns temp
- [ ] [verify] "Has my pH been stable in the last 24 hours?" → calls `get_probe_history`
- [ ] [verify] "Is everything in my tank normal?" → calls `summarize_tank_health`
- [ ] [verify] "Why did my ORP drop last night?" → calls `get_probe_history`, potentially `get_outlet_event_log`
- [ ] [verify] "Turn off my skimmer" → calls `control_outlet` with OFF state (with confirmation)
- [ ] [verify] "What changed in my tank yesterday afternoon?" → `get_outlet_event_log` + `get_probe_history`
- [ ] [verify] "Set up an alert for if my temp goes above 80°F" → calls `createAlert` (if implemented as writable tool — consider for Phase 6)

---

## Phase 5 Checklist Summary

- [ ] `mcp-go` dependency integrated
- [ ] All 8 tools implemented and tested
- [ ] Error handling returns clear messages
- [ ] Claude Desktop integration verified
- [ ] Claude Code integration verified
- [ ] All manual interaction scenarios pass

**Phase 5 is complete when:** Claude Desktop can answer "Is my tank healthy?" with real data from Symbiont, and "Turn off my return pump" physically toggles the outlet.
