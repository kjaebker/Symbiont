# Symbiont — Phase 3: CLI
> Cobra-based CLI with JSON output mode

**Deliverable:** The `symbiont` binary is installed and usable as a standalone tool. All subcommands return correct data in both human-readable and `--json` modes. Token management works end-to-end.

---

## 3.1 CLI Framework Setup

↳ depends on: Phase 2 complete

- [x] [code] Add dependency: `go get github.com/spf13/cobra`
- [x] [code] Create `cmd/symbiont/main.go`:
  - [x] Initialize root Cobra command
  - [x] Register all subcommand groups (probes, outlets, alerts, system, auth)
  - [x] Add persistent global flags:
    - [x] `--json` — output raw JSON (bool, default false)
    - [x] `--api-url` — override API base URL (string, default `http://localhost:8420`)
    - [x] `--token` — override auth token (string)
  - [x] Token resolution order: `--token` flag → `SYMBIONT_TOKEN` env var → `~/.config/symbiont/token` file
  - [x] If no token found and not running `auth tokens create`: error with helpful message
- [x] [code] Create `internal/cli/client.go`:
  - [x] `APIClient` struct wrapping `http.Client` with base URL and token
  - [x] `Get(ctx, path string, result any) error`
  - [x] `Put(ctx, path string, body any, result any) error`
  - [x] `Post(ctx, path string, body any, result any) error`
  - [x] `Delete(ctx, path string) error`
  - [x] All methods: set `Authorization: Bearer <token>` header, decode JSON response
  - [x] On non-2xx: decode error body and return as typed error
- [x] [code] Create `internal/cli/output.go`:
  - [x] `PrintJSON(v any)` — marshal and print with indentation
  - [x] `PrintTable(headers []string, rows [][]string)` — aligned column output
  - [x] `PrintKeyValue(pairs []KV)` — for single-record display
  - [x] `IsJSON(cmd *cobra.Command) bool` — reads `--json` flag from command
- [x] [verify] `go build ./cmd/symbiont` compiles

---

## 3.2 Probes Commands

- [x] [code] Create `internal/cli/probes.go`:
  - [x] `NewProbesCmd() *cobra.Command` — `symbiont probes`
  - [x] Sub-command `current`:
    - [x] Calls `GET /api/probes`
    - [x] Human output: table with PROBE, VALUE, UNIT, STATUS, UPDATED columns
    - [x] Status column color-coded (if terminal supports ANSI): green=normal, yellow=warning, red=critical
    - [x] JSON output: raw API response
  - [x] Sub-command `history <name>`:
    - [x] Flags: `--from`, `--to`, `--interval`
    - [x] Default `--from`: 24 hours ago (computed at runtime)
    - [x] Calls `GET /api/probes/<name>/history`
    - [x] Human output: table with TIMESTAMP, VALUE columns + summary stats (min, max, avg)
    - [x] JSON output: raw API response
- [x] [test] `internal/cli/cli_test.go`:
  - [x] Use `httptest.NewServer` to mock API
  - [x] Test API client methods (Get, Post, error handling, auth header)
- [ ] [verify] `symbiont probes current` shows real probe values from running API
- [ ] [verify] `symbiont probes current --json | jq .probes[0].value` returns a number
- [ ] [verify] `symbiont probes history Temp --interval 5m` returns data table

---

## 3.3 Outlets Commands

- [x] [code] Create `internal/cli/outlets.go`:
  - [x] `NewOutletsCmd() *cobra.Command` — `symbiont outlets`
  - [x] Sub-command `list`:
    - [x] Calls `GET /api/outlets`
    - [x] Human output: table with ID, NAME, STATE, TYPE columns
    - [x] State column color-coded: ON/AON=green, OFF/AOF=red, AUTO=blue
    - [x] JSON output: raw API response
  - [x] Sub-command `set <id> <ON|OFF|AUTO>`:
    - [x] Validates state arg is one of ON, OFF, AUTO (case-insensitive, normalized to uppercase)
    - [x] Calls `PUT /api/outlets/<id>`
    - [x] Human output: single-line confirmation `Outlet "Return Pump" set to OFF`
    - [x] JSON output: updated outlet object
    - [x] Error output: descriptive message if Apex rejects command
  - [x] Sub-command `events`:
    - [x] Calls `GET /api/outlets/events`
    - [x] Flags: `--outlet-id`, `--limit`
    - [x] Human output: table with ID, OUTLET, FROM, TO, BY, TIME columns
- [ ] [verify] `symbiont outlets list`
- [ ] [verify] `symbiont outlets set <id> OFF` — Apex outlet physically toggles
- [ ] [verify] `symbiont outlets set <id> AUTO` — returns to AUTO

---

## 3.4 Alerts Commands

- [x] [code] Create `internal/cli/alerts.go`:
  - [x] `NewAlertsCmd() *cobra.Command` — `symbiont alerts`
  - [x] Sub-command `list`:
    - [x] Calls `GET /api/alerts`
    - [x] Human output: table with ID, PROBE, CONDITION, THRESHOLD, SEVERITY, ENABLED columns
    - [x] JSON output: raw API response
  - [x] Sub-command `create`:
    - [x] Flags: `--probe`, `--condition`, `--low`, `--high`, `--severity`, `--cooldown`
    - [x] Calls `POST /api/alerts`
    - [x] Human output: `Alert rule #<id> created`
    - [x] JSON output: created rule object
  - [x] Sub-command `update <id>`:
    - [x] Same flags as create, all optional
    - [x] Calls `PUT /api/alerts/<id>`
  - [x] Sub-command `delete <id>`:
    - [x] Prompts for confirmation unless `--yes` flag provided
    - [x] Calls `DELETE /api/alerts/<id>`
    - [x] Human output: `Alert rule #<id> deleted`
- [ ] [verify] `symbiont alerts list` (may be empty initially)
- [ ] [verify] `symbiont alerts create --probe Temp --condition above --high 82 --severity warning`
- [ ] [verify] `symbiont alerts list` shows new rule
- [ ] [verify] `symbiont alerts delete <id>` removes rule

---

## 3.5 System Commands

- [x] [code] Create `internal/cli/system.go`:
  - [x] `NewSystemCmd() *cobra.Command` — `symbiont system`
  - [x] Sub-command `status`:
    - [x] Calls `GET /api/system`
    - [x] Human output: formatted key/value display (Controller, Poller, Database sections)
    - [x] JSON output: raw API response
  - [ ] Sub-command `backup`:
    - [ ] Calls `POST /api/system/backup`
    - [ ] Human output: progress indication, then backup path
    - [ ] Note: backup endpoint not yet implemented in API (Phase 6)
- [ ] [verify] `symbiont system status`

---

## 3.6 Auth Commands

- [x] [code] Create `internal/cli/auth.go`:
  - [x] `NewAuthCmd() *cobra.Command` — `symbiont auth`
  - [x] Sub-command `tokens list`:
    - [x] Calls `GET /api/tokens`
    - [x] Human output: table with ID, LABEL, CREATED, LAST USED columns
    - [x] Never shows token value
  - [x] Sub-command `tokens create`:
    - [x] Flag: `--label` (required)
    - [x] Calls `POST /api/tokens`
    - [x] Human output: shows token once with save prompt
    - [x] JSON output: `{ "id": 2, "label": "...", "token": "..." }`
  - [x] Sub-command `tokens revoke <id>`:
    - [x] Prompts confirmation unless `--yes`
    - [x] Calls `DELETE /api/tokens/<id>`
    - [x] Human output: `Token #<id> revoked`
  - [x] Special command `auth reset`:
    - [x] Flag: `--db-path` — directly opens SQLite (no API needed)
    - [x] Deletes all tokens, inserts a new default token, prints it
    - [x] Used for recovery when token is lost and API is inaccessible
    - [x] Requires `--yes` flag — destructive
- [ ] [verify] `symbiont auth tokens list`
- [ ] [verify] `symbiont auth tokens create --label "test"` → prints new token
- [ ] [verify] New token works for API requests
- [ ] [verify] `symbiont auth tokens revoke <id>` → token no longer works

---

## 3.7 Token Config File

- [x] [code] Create `internal/cli/token.go`:
  - [x] `LoadToken(flags *pflag.FlagSet) (string, error)` — implements resolution order:
    1. `--token` flag
    2. `SYMBIONT_TOKEN` environment variable
    3. `~/.config/symbiont/token` file (first line, trimmed)
    4. Returns error with helpful message if none found
  - [x] `SaveToken(token string) error` — writes to `~/.config/symbiont/token`, creates dirs if needed
- [x] [code] In `auth tokens create`: offer to save token to config file
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

- [x] Cobra CLI framework with persistent global flags
- [x] API client wrapper with token auth
- [x] Human-readable and JSON output modes on all commands
- [x] All command groups: probes, outlets, alerts, system, auth
- [x] Token config file for persistence
- [x] Auth reset command for recovery
- [ ] CLI installed on NixOS and shell completion working

**Phase 3 is complete when:** `symbiont probes current`, `symbiont outlets list`, `symbiont outlets set`, `symbiont system status` all work correctly from the terminal with no flags other than the initial token setup.
