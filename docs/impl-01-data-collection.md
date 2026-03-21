# Symbiont — Phase 1: Data Collection
> Apex client, Poller, DuckDB schema, NixOS service

**Deliverable:** The poller binary runs as a systemd service on NixOS, polls the Apex every 10 seconds, and DuckDB accumulates rows verifiable via the DuckDB CLI.

---

## 1.1 Reverse Engineering the Apex API

This must be completed before writing a single line of application code. The DevTools capture is ground truth for the AOS firmware version on your unit.

- [x] [research] Open Chrome DevTools → Network tab → filter by Fetch/XHR
- [x] [research] Navigate to Apex local IP in browser and log in normally
- [x] [research] Capture the login POST:
  - [x] Record exact URL (`POST /rest/login`)
  - [x] Record full request body (`{"login": "<user>", "password": "<pass>", "remember_me": true}`)
  - [x] Record response status code (200)
  - [x] Record `Set-Cookie` header value and cookie name (`connect.sid`)
  - [x] Record response body shape (`{"connect.sid": "<token>"}`)
- [x] [research] Capture a status GET after login:
  - [x] Record exact URL (`GET /rest/status`)
  - [x] Record request headers (`Cookie: connect.sid=<value>`)
  - [x] Save full response JSON to `testdata/status-response.json`
  - [x] Note all probe names, types, and value fields present (did, type, name, value — no unit field)
  - [x] Note all outlet fields present (did, ID, name, type, gid, status array, intensity)
  - [x] Note controller/system fields (serial, software, hardware, type, timezone, date)
  - [x] Note power events are in top-level `power` key (power.failed, power.restored as Unix epochs)
- [x] [research] Capture an outlet toggle PUT:
  - [x] Record exact URL (`PUT /rest/status/outputs/<did>`)
  - [x] Record request body (`{"did": "<did>", "status": ["ON", "", "OK", ""], "type": "outlet"}`)
  - [x] Record response body
  - [x] Confirm this is a runtime toggle that preserves outlet programs (vs `/rest/config/oconf/<did>` which overwrites programs)
- [x] [research] Test session expiry:
  - [x] Clearing cookies immediately triggers 401
  - [x] 401 response body: `{"username": ""}`
- [x] [decision] Trident data: N/A — no Trident connected to this unit
- [x] [decision] Vortech/WAV xstatus: N/A — no wireless devices connected to this unit
- [x] [code] Create `docs/apex-api-notes.md` documenting all findings
- [x] [code] Save representative sample responses as JSON fixtures in `testdata/` (status, config, ilog, link, things)

---

## 1.2 Repository Bootstrap

- [x] [code] Initialize Go module: `go mod init github.com/kjaebker/symbiont`
- [x] [code] Create directory structure per architecture doc:
  - [x] `cmd/poller/`, `cmd/api/`, `cmd/mcp/`, `cmd/symbiont/`
  - [x] `internal/apex/`, `internal/db/`, `internal/poller/`, `internal/api/`
  - [x] `internal/config/`, `internal/cli/`, `internal/mcp/`, `internal/alerts/`, `internal/notify/`
  - [x] `testdata/`, `docs/`
- [x] [code] Create `flake.nix` with Go dev environment:
  - [x] Go 1.23+ toolchain
  - [x] DuckDB CLI (`duckdb` binary)
  - [x] SQLite CLI (`sqlite3` binary)
  - [x] `gopls` and `delve` for IDE support
- [x] [code] Create `.env.example` with all config keys and comments (no real values)
- [x] [code] Create `.gitignore`: `.env`, `*.db`, `backups/`, `frontend/dist/`, binaries
- [x] [config] Create `.env` (not committed) with real Apex IP and credentials
- [x] [verify] `nix develop` enters the dev shell with all tools available
- [x] [verify] `go build ./...` compiles (empty packages are fine at this stage)

---

## 1.3 Configuration Package

- [x] [code] Create `internal/config/config.go`:
  - [x] Define `Config` struct with all fields matching architecture doc
  - [x] Implement `Load() *Config` using `os.Getenv` + `godotenv` for `.env` file
  - [x] Add `go get github.com/joho/godotenv`
  - [x] Required fields: `ApexURL`, `ApexUser`, `ApexPass`
  - [x] Optional fields with defaults: `DBPath`, `SQLitePath`, `PollInterval`, `APIPort`, `LogLevel`
  - [x] Duration parsing for `PollInterval` (`time.ParseDuration`)
  - [x] Validate required fields at load time — `log.Fatal` if missing
- [x] [test] Write `internal/config/config_test.go`:
  - [x] Test defaults are applied for optional fields
  - [x] Test required fields cause fatal when missing
  - [x] Test duration parsing for poll interval
- [x] [verify] `go test ./internal/config/...` passes

---

## 1.4 Apex Client

↳ depends on: 1.1 (DevTools capture complete)

- [x] [code] Create `internal/apex/models.go`:
  - [x] `StatusResponse` struct with `System`, `Inputs []Input`, `Outputs []Output`, `Modules []Module`, `Feed FeedStatus`, `Power PowerInfo`, `Link LinkInfo`, `Nstat NetworkStatus`
  - [x] `SystemInfo` struct: serial, hostname, software (not firmware), hardware, type, timezone (string), date (Unix epoch int64), extra
  - [x] `PowerInfo` struct: failed (int64 Unix epoch), restored (int64 Unix epoch) — top-level, NOT in system
  - [x] `Input` struct: did, name, value (float64), type (no unit field — type serves as unit indicator: "Temp", "pH", "Amps", "pwr", "volts", "digital")
  - [x] `Output` struct: did (string), ID (int), name, type ("outlet"/"variable"/"alert"/"virtual"/"serial"/"24v"), gid, status ([]string — 4 elements: [state, intensity_or_empty, health, unknown]), intensity (int, optional)
  - [x] `OutletState` type: `ON`, `OFF`, `AON`, `AOF` constants (no bare "AUTO" — use AON/AOF to return to auto mode)
  - [x] `FeedStatus` struct: name (int), active (int)
  - [x] **Field names must match DevTools capture exactly** — see `docs/apex-api-notes.md`
- [x] [code] Create `internal/apex/client.go`:
  - [x] Define `Client` interface with `Status`, `Outlets`, `SetOutlet` methods
  - [x] Implement `client` struct: baseURL, username, password, `*http.Client`, `sync.Mutex`, cookie
  - [x] Implement `NewClient(baseURL, user, pass string) (Client, error)` — creates client and authenticates immediately
  - [x] Implement `login(ctx context.Context) error`:
    - [x] POST to `/rest/login` with correct body (from DevTools capture)
    - [x] Extract `connect.sid` cookie (or whatever cookie name DevTools showed)
    - [x] Store cookie on client struct under mutex
    - [x] Return error if status is not 200
  - [x] Implement `do(ctx context.Context, req *http.Request) (*http.Response, error)`:
    - [x] Attach session cookie to request
    - [x] Execute request
    - [x] If 401 response: call `login()`, retry request once
    - [x] Return response or error
  - [x] Implement `Status(ctx context.Context) (*StatusResponse, error)`:
    - [x] Build GET request to `/rest/status`
    - [x] Call `do()`, decode JSON response into `StatusResponse`
  - [x] Implement `SetOutlet(ctx context.Context, did string, state OutletState) error`:
    - [x] Build PUT request to `/rest/status/outputs/<did>` with body `{"did": "<did>", "status": ["<state>", "", "OK", ""], "type": "outlet"}`
    - [x] Call `do()`, decode response
  - [x] Set `http.Client` timeout: 10 seconds
  - [x] Add `go get` for no external deps needed (stdlib only for HTTP)
- [x] [code] Create `internal/apex/parser.go`:
  - [x] `NormalizeProbeType(input Input) string` — maps Apex type strings to canonical types (`Temp`, `pH`, `Amps`, `pwr`, `volts`, `digital`)
  - [x] `ParsePowerEvents(power PowerInfo) []PowerEvent` — parses power.failed/power.restored Unix epoch timestamps into typed events
  - [x] `CorrelateOutletPower(inputs []Input, outputs []Output)` — matches per-outlet amp/watt input entries to outlets by name convention (`<Name>A` for amps, `<Name>W` for watts)
- [x] [test] Create `testdata/status-response.json` with real sample from DevTools
- [x] [test] Create `internal/apex/client_test.go`:
  - [x] Use `httptest.NewServer` to mock the Apex
  - [x] Test successful login and status fetch
  - [x] Test 401 triggers re-auth and retry
  - [x] Test `SetOutlet` sends correct body
  - [x] Test timeout behavior
- [x] [test] Create `internal/apex/parser_test.go`:
  - [x] Test probe type normalization
  - [x] Test power event parsing with real sample timestamps
- [x] [verify] `go test ./internal/apex/...` passes

---

## 1.5 DuckDB Package

- [x] [code] Add dependency: `go get github.com/marcboeker/go-duckdb`
- [x] [verify] `go-duckdb` compiles correctly on NixOS (CGO dependency — test early)
  - [x] [!] If CGO fails on NixOS: investigate `nix-ld` or pkg-config setup, document fix in `docs/nixos-notes.md`
- [x] [code] Create `internal/db/schema.go`:
  - [x] `CreateSchema(db *sql.DB) error` — idempotent, uses `CREATE TABLE IF NOT EXISTS`
  - [x] All four DuckDB tables from architecture doc: `probe_readings`, `outlet_states`, `power_events`, `controller_meta`
  - [x] `MigrateSchema(db *sql.DB) error` — placeholder for future schema migrations (runs after CreateSchema)
- [x] [code] Create `internal/db/duckdb.go`:
  - [x] `Open(path string) (*DuckDB, error)` — opens read-write connection, runs schema creation
  - [x] `OpenReadOnly(path string) (*DuckDB, error)` — opens read-only connection
  - [x] `DuckDB` struct wrapping `*sql.DB`
  - [x] `WriteProbeReadings(ctx, ts time.Time, inputs []apex.Input) error` — batch INSERT
  - [x] `WriteOutletStates(ctx, ts time.Time, outputs []apex.Output) error` — batch INSERT
  - [x] `WritePowerEvents(ctx, ts time.Time, power apex.PowerInfo) error` — deduplicating INSERT
  - [x] `WriteControllerMeta(ctx, ts time.Time, sys apex.SystemInfo) error` — INSERT
  - [x] `WritePollCycle(ctx, ts time.Time, status *apex.StatusResponse) error` — wraps all writes in single transaction
  - [x] `Close() error`
- [x] [code] Create `internal/db/queries.go` (read queries, used by API server later):
  - [x] `CurrentProbeReadings(ctx) ([]ProbeReading, error)` — latest value per probe
  - [x] `ProbeHistory(ctx, name string, from, to time.Time, interval string) ([]DataPoint, error)` — bucketed time-series
  - [x] `CurrentOutletStates(ctx) ([]OutletState, error)` — latest state per outlet
  - [x] `ControllerMeta(ctx) (*ControllerMeta, error)` — most recent controller snapshot
  - [x] `LastPollTime(ctx) (time.Time, error)`
- [x] [code] Define internal result types in `internal/db/models.go`:
  - [x] `ProbeReading`, `OutletState`, `DataPoint`, `ControllerMeta`
- [x] [test] Create `internal/db/duckdb_test.go`:
  - [x] Use temp file for test DB (cleaned up with `t.Cleanup`)
  - [x] Test schema creation is idempotent
  - [x] Test `WritePollCycle` with sample data
  - [x] Test transaction rollback on partial write failure
  - [x] Test `CurrentProbeReadings` returns latest row per probe
  - [x] Test `ProbeHistory` bucketing
- [x] [verify] `go test ./internal/db/...` passes

---

## 1.6 Poller Binary

↳ depends on: 1.3, 1.4, 1.5

- [x] [code] Create `internal/poller/poller.go`:
  - [x] `Poller` struct: apex client, DuckDB, interval, logger
  - [x] `New(apex apex.Client, db *db.DuckDB, interval time.Duration, logger *slog.Logger) *Poller`
  - [x] `Run(ctx context.Context)` — ticker loop, calls `poll()` immediately then on each tick
  - [x] `poll(ctx context.Context)` — calls `apex.Status()`, calls `db.WritePollCycle()`, logs result
  - [x] Skip cycle (log + return) on Apex error — no crash
  - [x] Skip cycle (log + return) on DuckDB write error — no crash
  - [x] Log structured output per cycle: duration_ms, probes count, outlets count
- [x] [code] Create `cmd/poller/main.go`:
  - [x] Load config
  - [x] Set up `slog` JSON logger
  - [x] Open DuckDB (read-write)
  - [x] Create Apex client (login on startup)
  - [x] Create and run Poller
  - [x] Handle SIGTERM/SIGINT via `signal.NotifyContext`
  - [x] Graceful shutdown: wait for in-flight poll to complete
- [x] [verify] `go build ./cmd/poller` compiles
- [x] [verify] Run `./poller` locally against real Apex, watch logs
- [x] [verify] After 60 seconds, open DuckDB CLI and confirm rows exist:
  ```sql
  SELECT COUNT(*), probe_name FROM probe_readings GROUP BY probe_name;
  SELECT COUNT(*), outlet_id  FROM outlet_states  GROUP BY outlet_id;
  ```
- [x] [verify] Kill poller with SIGTERM — confirm clean shutdown in logs
- [x] [verify] Restart poller — confirm it resumes without errors or schema conflicts

---

## 1.7 Structured Logging

- [ ] [code] Set up `log/slog` with JSON handler in each binary's `main.go`
- [ ] [code] Log level controlled by `SYMBIONT_LOG_LEVEL` env var (default: `info`)
- [ ] [code] All log lines in Poller include: `service=poller`, `ts`, structured fields
- [ ] [verify] Logs are valid JSON (`journalctl -u symbiont-poller | jq .`)

---

## 1.8 NixOS Systemd Service

↳ depends on: 1.6 complete and verified

- [ ] [config] Create `symbiont` system user and group in `flake.nix` (or `configuration.nix`)
- [ ] [config] Create `/var/lib/symbiont/` with correct ownership
- [ ] [config] Create `/etc/symbiont/env` with real values, mode 0400, owned by `symbiont`
- [ ] [config] Add `symbiont-poller` systemd service definition to `flake.nix`:
  - [ ] `ExecStart` pointing to poller binary in Nix store
  - [ ] `EnvironmentFile = /etc/symbiont/env`
  - [ ] `Restart = always`, `RestartSec = 5s`
  - [ ] `User = symbiont`, `Group = symbiont`
  - [ ] `StateDirectory = symbiont`
  - [ ] Hardening: `PrivateTmp`, `NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths`
- [ ] [verify] `sudo systemctl start symbiont-poller`
- [ ] [verify] `sudo systemctl status symbiont-poller` → active (running)
- [ ] [verify] `sudo journalctl -u symbiont-poller -f` → JSON log lines every 10s
- [ ] [verify] `sudo systemctl stop symbiont-poller` → clean shutdown
- [ ] [verify] After restart: DB continues accumulating without gaps
- [ ] [config] Enable service at boot: `wantedBy = [ "multi-user.target" ]`
- [ ] [verify] Reboot mini PC → poller starts automatically

---

## 1.9 Retention and Cleanup (stub)

- [ ] [code] Create stub `internal/db/cleanup.go` with `DeleteOldRows(ctx, retentionDays int) error`
  - [ ] Deletes rows from all four DuckDB tables older than `retentionDays`
  - [ ] Returns count of deleted rows per table
  - [ ] Not wired to a timer yet — that happens in Phase 6
- [ ] [test] Test that `DeleteOldRows` removes expected rows and leaves recent rows

---

## Phase 1 Checklist Summary

- [ ] DevTools capture complete and documented
- [ ] Repository initialized with correct structure
- [ ] Config package loading from env
- [ ] Apex client with session management and 401 retry
- [ ] DuckDB schema and write functions
- [ ] Poller binary running and verified locally
- [ ] Poller running as systemd service on NixOS
- [ ] Data accumulating in DuckDB, verifiable via CLI

**Phase 1 is complete when:** `duckdb /var/lib/symbiont/telemetry.db "SELECT COUNT(*) FROM probe_readings"` returns a growing row count with the service running.
