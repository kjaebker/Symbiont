# Symbiont — Phase 3: CLI
> Cobra-based CLI with JSON output mode

**Deliverable:** The `symbiont` binary is installed and usable as a standalone tool. All subcommands return correct data in both human-readable and `--json` modes. Token management works end-to-end.

---

## 3.1 CLI Framework Setup

↳ depends on: Phase 2 complete

- [ ] [code] Add dependency: `go get github.com/spf13/cobra`
- [ ] [code] Create `cmd/symbiont/main.go`:
  - [ ] Initialize root Cobra command
  - [ ] Register all subcommand groups (probes, outlets, alerts, system, auth)
  - [ ] Add persistent global flags:
    - [ ] `--json` — output raw JSON (bool, default false)
    - [ ] `--api-url` — override API base URL (string, default `http://localhost:8420`)
    - [ ] `--token` — override auth token (string)
  - [ ] Token resolution order: `--token` flag → `SYMBIONT_TOKEN` env var → `~/.config/symbiont/token` file
  - [ ] If no token found and not running `auth tokens create`: error with helpful message
- [ ] [code] Create `internal/cli/client.go`:
  - [ ] `APIClient` struct wrapping `http.Client` with base URL and token
  - [ ] `Get(ctx, path string, result any) error`
  - [ ] `Put(ctx, path string, body any, result any) error`
  - [ ] `Post(ctx, path string, body any, result any) error`
  - [ ] `Delete(ctx, path string) error`
  - [ ] All methods: set `Authorization: Bearer <token>` header, decode JSON response
  - [ ] On non-2xx: decode error body and return as typed error
- [ ] [code] Create `internal/cli/output.go`:
  - [ ] `PrintJSON(v any)` — marshal and print with indentation
  - [ ] `PrintTable(headers []string, rows [][]string)` — aligned column output
  - [ ] `PrintKeyValue(pairs []KV)` — for single-record display
  - [ ] `IsJSON(cmd *cobra.Command) bool` — reads `--json` flag from command
- [ ] [verify] `go build ./cmd/symbiont` compiles

---

## 3.2 Probes Commands

- [ ] [code] Create `internal/cli/probes.go`:
  - [ ] `NewProbesCmd() *cobra.Command` — `symbiont probes`
  - [ ] Sub-command `current`:
    - [ ] Calls `GET /api/probes`
    - [ ] Human output: table with PROBE, VALUE, UNIT, STATUS, UPDATED columns
    - [ ] Status column color-coded (if terminal supports ANSI): green=normal, yellow=warning, red=critical
    - [ ] JSON output: raw API response
  - [ ] Sub-command `history <name>`:
    - [ ] Flags: `--from`, `--to`, `--interval`
    - [ ] Default `--from`: 24 hours ago (computed at runtime)
    - [ ] Calls `GET /api/probes/<name>/history`
    - [ ] Human output: table with TIMESTAMP, VALUE columns + summary stats (min, max, avg)
    - [ ] JSON output: raw API response
- [ ] [test] `internal/cli/probes_test.go`:
  - [ ] Use `httptest.NewServer` to mock API
  - [ ] Test `current` human output columns
  - [ ] Test `current --json` raw output
  - [ ] Test `history` with valid probe name
  - [ ] Test `history` with unknown probe name (API returns 404)
- [ ] [verify] `symbiont probes current` shows real probe values from running API
- [ ] [verify] `symbiont probes current --json | jq .probes[0].value` returns a number
- [ ] [verify] `symbiont probes history Temp --interval 5m` returns data table

---

## 3.3 Outlets Commands

- [ ] [code] Create `internal/cli/outlets.go`:
  - [ ] `NewOutletsCmd() *cobra.Command` — `symbiont outlets`
  - [ ] Sub-command `list`:
    - [ ] Calls `GET /api/outlets`
    - [ ] Human output: table with DID, NAME, STATE, TYPE, HEALTH, WATTS, AMPS columns
    - [ ] State from `status[0]}`, health from `status[2]` ("OK" or "---")
    - [ ] WATTS/AMPS correlated from input entries by name convention (may be empty for non-EB832 outlets)
    - [ ] State column color-coded: ON/AON=green, OFF/AOF=red, AUTO=blue
    - [ ] JSON output: raw API response
  - [ ] Sub-command `set <id> <ON|OFF|AUTO>`:
    - [ ] Validates state arg is one of ON, OFF, AUTO (case-insensitive, normalized to uppercase)
    - [ ] Calls `PUT /api/outlets/<id>`
    - [ ] Human output: single-line confirmation `Outlet "Return Pump" set to OFF`
    - [ ] JSON output: updated outlet object
    - [ ] Error output: descriptive message if Apex rejects command
- [ ] [test]:
  - [ ] Test `list` output shape
  - [ ] Test `set` with valid state
  - [ ] Test `set` with invalid state arg
- [ ] [verify] `symbiont outlets list`
- [ ] [verify] `symbiont outlets set <id> OFF` — Apex outlet physically toggles
- [ ] [verify] `symbiont outlets set <id> AUTO` — returns to AUTO

---

## 3.4 Alerts Commands

- [ ] [code] Create `internal/cli/alerts.go`:
  - [ ] `NewAlertsCmd() *cobra.Command` — `symbiont alerts`
  - [ ] Sub-command `list`:
    - [ ] Calls `GET /api/alerts`
    - [ ] Human output: table with ID, PROBE, CONDITION, THRESHOLD, SEVERITY, ENABLED columns
    - [ ] JSON output: raw API response
  - [ ] Sub-command `create`:
    - [ ] Flags: `--probe`, `--condition`, `--low`, `--high`, `--severity`, `--cooldown`
    - [ ] Interactive prompts if flags not provided (use `bufio.Scanner` for simplicity)
    - [ ] Calls `POST /api/alerts`
    - [ ] Human output: `Alert rule #<id> created`
    - [ ] JSON output: created rule object
  - [ ] Sub-command `update <id>`:
    - [ ] Same flags as create, all optional
    - [ ] Calls `PUT /api/alerts/<id>`
  - [ ] Sub-command `delete <id>`:
    - [ ] Prompts for confirmation unless `--yes` flag provided
    - [ ] Calls `DELETE /api/alerts/<id>`
    - [ ] Human output: `Alert rule #<id> deleted`
  - [ ] Sub-command `events`:
    - [ ] Calls `GET /api/alerts/events` (add this endpoint to API server in Phase 6 if not present)
    - [ ] Human output: table of recent firings
- [ ] [verify] `symbiont alerts list` (may be empty initially)
- [ ] [verify] `symbiont alerts create --probe Temp --condition above --high 82 --severity warning`
- [ ] [verify] `symbiont alerts list` shows new rule
- [ ] [verify] `symbiont alerts delete <id>` removes rule

---

## 3.5 System Commands

- [ ] [code] Create `internal/cli/system.go`:
  - [ ] `NewSystemCmd() *cobra.Command` — `symbiont system`
  - [ ] Sub-command `status`:
    - [ ] Calls `GET /api/system`
    - [ ] Human output: formatted key/value display:
      ```
      Controller
        Serial:    AC5:12345
        Firmware:  5.08A_7A18

      Poller
        Last poll: 3 seconds ago
        Status:    OK
        Interval:  10s

      Database
        DuckDB:    128 MB
        SQLite:    1.0 MB
      ```
    - [ ] JSON output: raw API response
  - [ ] Sub-command `backup`:
    - [ ] Calls `POST /api/system/backup`
    - [ ] Human output: progress indication, then `Backup saved to /var/lib/symbiont/backups/telemetry-2025-03-20.db`
    - [ ] JSON output: backup job result
- [ ] [verify] `symbiont system status`
- [ ] [verify] `symbiont system backup` creates file in backup directory

---

## 3.6 Auth Commands

- [ ] [code] Create `internal/cli/auth.go`:
  - [ ] `NewAuthCmd() *cobra.Command` — `symbiont auth`
  - [ ] Sub-command `tokens list`:
    - [ ] Calls `GET /api/auth/tokens`
    - [ ] Human output: table with ID, LABEL, CREATED, LAST USED columns
    - [ ] Never shows token value
  - [ ] Sub-command `tokens create`:
    - [ ] Flag: `--label` (required)
    - [ ] Calls `POST /api/auth/tokens`
    - [ ] Human output:
      ```
      Token created (save this — shown once):
      a3f8e2c1d7b4...
      ```
    - [ ] JSON output: `{ "id": 2, "label": "claude-desktop", "token": "..." }`
  - [ ] Sub-command `tokens revoke <id>`:
    - [ ] Prompts confirmation unless `--yes`
    - [ ] Calls `DELETE /api/auth/tokens/<id>`
    - [ ] Human output: `Token #<id> revoked`
  - [ ] Special command `auth reset`:
    - [ ] Flag: `--db-path` — directly opens SQLite (no API needed)
    - [ ] Deletes all tokens, inserts a new default token, prints it
    - [ ] Used for recovery when token is lost and API is inaccessible
    - [ ] Requires `--yes` flag — destructive
- [ ] [verify] `symbiont auth tokens list`
- [ ] [verify] `symbiont auth tokens create --label "test"` → prints new token
- [ ] [verify] New token works for API requests
- [ ] [verify] `symbiont auth tokens revoke <id>` → token no longer works

---

## 3.7 Token Config File

- [ ] [code] Create `internal/cli/token.go`:
  - [ ] `LoadToken(flags *pflag.FlagSet) (string, error)` — implements resolution order:
    1. `--token` flag
    2. `SYMBIONT_TOKEN` environment variable
    3. `~/.config/symbiont/token` file (first line, trimmed)
    4. Returns error with helpful message if none found
  - [ ] `SaveToken(token string) error` — writes to `~/.config/symbiont/token`, creates dirs if needed
- [ ] [code] In `auth tokens create`: offer to save token to config file:
  ```
  Save token to ~/.config/symbiont/token? [y/N]
  ```
- [ ] [verify] Save token to config file → subsequent commands work without `--token` flag

---

## 3.8 NixOS CLI Installation

- [ ] [config] Add `symbiont` CLI binary to NixOS environment packages in `flake.nix`
- [ ] [verify] `symbiont --help` works from any directory
- [ ] [verify] `symbiont probes current` works without any flags (token from config file or env)
- [ ] [config] Add shell completion setup to `flake.nix` (Cobra provides this for bash/zsh/fish):
  - [ ] `symbiont completion bash > /etc/bash_completion.d/symbiont`

---

## Phase 3 Checklist Summary

- [ ] Cobra CLI framework with persistent global flags
- [ ] API client wrapper with token auth
- [ ] Human-readable and JSON output modes on all commands
- [ ] All command groups: probes, outlets, alerts, system, auth
- [ ] Token config file for persistence
- [ ] Auth reset command for recovery
- [ ] CLI installed on NixOS and shell completion working

**Phase 3 is complete when:** `symbiont probes current`, `symbiont outlets list`, `symbiont outlets set`, `symbiont system status` all work correctly from the terminal with no flags other than the initial token setup.
