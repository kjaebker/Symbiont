# Symbiont — Phase 1: Data Collection
> Apex client, Poller, DuckDB schema, NixOS service

**Deliverable:** The poller binary runs as a systemd service on NixOS, polls the Apex every 10 seconds, and DuckDB accumulates rows verifiable via the DuckDB CLI.

---

## 1.1 Reverse Engineering the Apex API

This must be completed before writing a single line of application code. The DevTools capture is ground truth for the AOS firmware version on your unit.

- [ ] [research] Open Chrome DevTools → Network tab → filter by Fetch/XHR
- [ ] [research] Navigate to Apex local IP in browser and log in normally
- [ ] [research] Capture the login POST:
  - [ ] Record exact URL (likely `http://<apex-ip>/rest/login`)
  - [ ] Record full request body (JSON shape, field names, encoding)
  - [ ] Record response status code
  - [ ] Record `Set-Cookie` header value and cookie name (likely `connect.sid`)
  - [ ] Record response body shape if any
- [ ] [research] Capture a status GET after login:
  - [ ] Record exact URL (`/rest/status` or variant)
  - [ ] Record request headers (Cookie header, any others)
  - [ ] Save full response JSON to `docs/apex-status-sample.json`
  - [ ] Note all probe names, types, and value fields present
  - [ ] Note all outlet fields present (id/DID, name, state, xstatus, watts, amps)
  - [ ] Note controller/system fields (serial, firmware, power timestamps)
- [ ] [research] Capture an outlet toggle PUT:
  - [ ] Record exact URL (`/rest/outlets/<id>` — note ID format)
  - [ ] Record request body (JSON shape, state field name and value format)
  - [ ] Record response body
- [ ] [research] Test session expiry:
  - [ ] Note how long until a 401 is returned naturally, or force one by clearing cookies
  - [ ] Confirm the 401 response body/shape
- [ ] [decision] Does Trident data appear in `/rest/status`? Record Ca, Alk, Mg field locations if present
- [ ] [decision] Do Vortech/WAV xstatus fields appear? Record their shape
- [x] [code] Create `docs/apex-api-notes.md` documenting all findings
- [ ] [code] Save representative sample responses as JSON fixtures in `testdata/`

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
- [ ] [config] Create `.env` (not committed) with real Apex IP and credentials
- [ ] [verify] `nix develop` enters the dev shell with all tools available
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

- [ ] [code] Create `internal/apex/models.go`:
  - [ ] `StatusResponse` struct with `System`, `Inputs []Input`, `Outputs []Output`
  - [ ] `SystemInfo` struct: serial, hostname, firmware, hardware, power_failed, power_restored, date
  - [ ] `Input` struct: name, value (float64), unit, type
  - [ ] `Output` struct: DID/id, name, state, xstatus, watts, amps
  - [ ] `OutletState` type: `ON`, `OFF`, `AUTO` constants
  - [ ] **Field names must match DevTools capture exactly** — adjust from architecture doc defaults if needed
- [ ] [code] Create `internal/apex/client.go`:
  - [ ] Define `Client` interface with `Status`, `Outlets`, `SetOutlet` methods
  - [ ] Implement `client` struct: baseURL, username, password, `*http.Client`, `sync.Mutex`, cookie
  - [ ] Implement `NewClient(baseURL, user, pass string) (Client, error)` — creates client and authenticates immediately
  - [ ] Implement `login(ctx context.Context) error`:
    - [ ] POST to `/rest/login` with correct body (from DevTools capture)
    - [ ] Extract `connect.sid` cookie (or whatever cookie name DevTools showed)
    - [ ] Store cookie on client struct under mutex
    - [ ] Return error if status is not 200
  - [ ] Implement `do(ctx context.Context, req *http.Request) (*http.Response, error)`:
    - [ ] Attach session cookie to request
    - [ ] Execute request
    - [ ] If 401 response: call `login()`, retry request once
    - [ ] Return response or error
  - [ ] Implement `Status(ctx context.Context) (*StatusResponse, error)`:
    - [ ] Build GET request to `/rest/status`
    - [ ] Call `do()`, decode JSON response into `StatusResponse`
  - [ ] Implement `SetOutlet(ctx context.Context, id string, state OutletState) (*Output, error)`:
    - [ ] Build PUT request to `/rest/outlets/<id>` with correct body shape
    - [ ] Call `do()`, decode response
  - [ ] Set `http.Client` timeout: 10 seconds
  - [ ] Add `go get` for no external deps needed (stdlib only for HTTP)
- [ ] [code] Create `internal/apex/parser.go`:
  - [ ] `NormalizeProbeType(input Input) string` — maps Apex type strings to canonical types (`temp`, `pH`, `ORP`, `salinity`)
  - [ ] `ParsePowerEvents(system SystemInfo) []PowerEvent` — parses power_failed/power_restored timestamps into typed events
- [ ] [test] Create `testdata/status-response.json` with real sample from DevTools
- [ ] [test] Create `internal/apex/client_test.go`:
  - [ ] Use `httptest.NewServer` to mock the Apex
  - [ ] Test successful login and status fetch
  - [ ] Test 401 triggers re-auth and retry
  - [ ] Test `SetOutlet` sends correct body
  - [ ] Test timeout behavior
- [ ] [test] Create `internal/apex/parser_test.go`:
  - [ ] Test probe type normalization
  - [ ] Test power event parsing with real sample timestamps
- [ ] [verify] `go test ./internal/apex/...` passes

---

## 1.5 DuckDB Package

- [ ] [code] Add dependency: `go get github.com/marcboeker/go-duckdb`
- [ ] [verify] `go-duckdb` compiles correctly on NixOS (CGO dependency — test early)
  - [ ] [!] If CGO fails on NixOS: investigate `nix-ld` or pkg-config setup, document fix in `docs/nixos-notes.md`
- [ ] [code] Create `internal/db/schema.go`:
  - [ ] `CreateSchema(db *sql.DB) error` — idempotent, uses `CREATE TABLE IF NOT EXISTS`
  - [ ] All four DuckDB tables from architecture doc: `probe_readings`, `outlet_states`, `power_events`, `controller_meta`
  - [ ] `MigrateSchema(db *sql.DB) error` — placeholder for future schema migrations (runs after CreateSchema)
- [ ] [code] Create `internal/db/duckdb.go`:
  - [ ] `Open(path string) (*DuckDB, error)` — opens read-write connection, runs schema creation
  - [ ] `OpenReadOnly(path string) (*DuckDB, error)` — opens read-only connection
  - [ ] `DuckDB` struct wrapping `*sql.DB`
  - [ ] `WriteProbeReadings(ctx, ts time.Time, inputs []apex.Input) error` — batch INSERT
  - [ ] `WriteOutletStates(ctx, ts time.Time, outputs []apex.Output) error` — batch INSERT
  - [ ] `WritePowerEvents(ctx, ts time.Time, sys apex.SystemInfo) error` — deduplicating INSERT
  - [ ] `WriteControllerMeta(ctx, ts time.Time, sys apex.SystemInfo) error` — INSERT
  - [ ] `WritePollCycle(ctx, ts time.Time, status *apex.StatusResponse) error` — wraps all writes in single transaction
  - [ ] `Close() error`
- [ ] [code] Create `internal/db/queries.go` (read queries, used by API server later):
  - [ ] `CurrentProbeReadings(ctx) ([]ProbeReading, error)` — latest value per probe
  - [ ] `ProbeHistory(ctx, name string, from, to time.Time, interval string) ([]DataPoint, error)` — bucketed time-series
  - [ ] `CurrentOutletStates(ctx) ([]OutletState, error)` — latest state per outlet
  - [ ] `ControllerMeta(ctx) (*ControllerMeta, error)` — most recent controller snapshot
  - [ ] `LastPollTime(ctx) (time.Time, error)`
- [ ] [code] Define internal result types in `internal/db/models.go`:
  - [ ] `ProbeReading`, `OutletState`, `DataPoint`, `ControllerMeta`
- [ ] [test] Create `internal/db/duckdb_test.go`:
  - [ ] Use temp file for test DB (cleaned up with `t.Cleanup`)
  - [ ] Test schema creation is idempotent
  - [ ] Test `WritePollCycle` with sample data
  - [ ] Test transaction rollback on partial write failure
  - [ ] Test `CurrentProbeReadings` returns latest row per probe
  - [ ] Test `ProbeHistory` bucketing
- [ ] [verify] `go test ./internal/db/...` passes

---

## 1.6 Poller Binary

↳ depends on: 1.3, 1.4, 1.5

- [ ] [code] Create `internal/poller/poller.go`:
  - [ ] `Poller` struct: apex client, DuckDB, interval, logger
  - [ ] `New(apex apex.Client, db *db.DuckDB, interval time.Duration, logger *slog.Logger) *Poller`
  - [ ] `Run(ctx context.Context)` — ticker loop, calls `poll()` immediately then on each tick
  - [ ] `poll(ctx context.Context)` — calls `apex.Status()`, calls `db.WritePollCycle()`, logs result
  - [ ] Skip cycle (log + return) on Apex error — no crash
  - [ ] Skip cycle (log + return) on DuckDB write error — no crash
  - [ ] Log structured output per cycle: duration_ms, probes count, outlets count
- [ ] [code] Create `cmd/poller/main.go`:
  - [ ] Load config
  - [ ] Set up `slog` JSON logger
  - [ ] Open DuckDB (read-write)
  - [ ] Create Apex client (login on startup)
  - [ ] Create and run Poller
  - [ ] Handle SIGTERM/SIGINT via `signal.NotifyContext`
  - [ ] Graceful shutdown: wait for in-flight poll to complete
- [ ] [verify] `go build ./cmd/poller` compiles
- [ ] [verify] Run `./poller` locally against real Apex, watch logs
- [ ] [verify] After 60 seconds, open DuckDB CLI and confirm rows exist:
  ```sql
  SELECT COUNT(*), probe_name FROM probe_readings GROUP BY probe_name;
  SELECT COUNT(*), outlet_id  FROM outlet_states  GROUP BY outlet_id;
  ```
- [ ] [verify] Kill poller with SIGTERM — confirm clean shutdown in logs
- [ ] [verify] Restart poller — confirm it resumes without errors or schema conflicts

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
