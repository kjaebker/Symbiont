# Symbiont — Phase 2: API Server
> Go HTTP server, SQLite, auth, SSE

**Deliverable:** Full REST API queryable via curl. All probe and outlet endpoints return real data. Outlet control sends commands to the Apex and logs the event. SSE stream pushes updates every 10 seconds.

---

## 2.1 SQLite Package

- [x] [code] Add dependency: `go get modernc.org/sqlite`
  - [x] Prefer `modernc.org/sqlite` (pure Go, no CGO) over `mattn/go-sqlite3`
- [x] [code] Create `internal/db/sqlite_schema.go`:
  - [x] `CreateSQLiteSchema(db *sql.DB) error` — idempotent, `CREATE TABLE IF NOT EXISTS`
  - [x] All tables from architecture doc: `auth_tokens`, `probe_config`, `outlet_config`, `alert_rules`, `notification_targets`, `alert_events`, `outlet_event_log`, `backup_jobs`
  - [x] All indexes from architecture doc
  - [x] `PRAGMA journal_mode=WAL` and `PRAGMA foreign_keys=ON` on open
- [x] [code] Create `internal/db/sqlite.go`:
  - [x] `OpenSQLite(path string) (*SQLiteDB, error)` — opens DB, runs schema, runs PRAGMAs
  - [x] `SQLiteDB` struct wrapping `*sql.DB`
  - [x] `Close() error`
- [x] [code] Create `internal/db/sqlite_queries.go` with typed query functions:
  - [x] `ValidateToken(token string) (bool, int64)` — returns valid + token ID
  - [x] `TouchToken(id int64) error` — updates `last_used`
  - [x] `InsertToken(label string) (string, error)` — generates 32-byte random token, inserts, returns token string
  - [x] `ListTokens() ([]AuthToken, error)`
  - [x] `DeleteToken(id int64) error`
  - [x] `GetProbeConfig(probeName string) (*ProbeConfig, error)`
  - [x] `ListProbeConfigs() ([]ProbeConfig, error)`
  - [x] `UpsertProbeConfig(cfg ProbeConfig) error`
  - [x] `GetOutletConfig(outletID string) (*OutletConfig, error)`
  - [x] `ListOutletConfigs() ([]OutletConfig, error)`
  - [x] `UpsertOutletConfig(cfg OutletConfig) error`
  - [x] `ListEnabledAlertRules() ([]AlertRule, error)`
  - [x] `InsertAlertRule(rule AlertRule) (int64, error)`
  - [x] `UpdateAlertRule(id int64, rule AlertRule) error`
  - [x] `DeleteAlertRule(id int64) error`
  - [x] `InsertOutletEvent(e OutletEvent) error`
  - [x] `ListOutletEvents(outletID string, limit int) ([]OutletEvent, error)`
  - [x] `InsertBackupJob(job BackupJob) (int64, error)`
  - [x] `UpdateBackupJob(id int64, status string, err string) error`
  - [x] `ListBackupJobs(limit int) ([]BackupJob, error)`
- [x] [code] Define SQLite result types in `internal/db/sqlite_models.go`:
  - [x] `AuthToken`, `ProbeConfig`, `OutletConfig`, `AlertRule`, `OutletEvent`, `BackupJob`
- [x] [test] Create `internal/db/sqlite_test.go`:
  - [x] Use `:memory:` for test DB
  - [x] Test schema creation idempotency
  - [x] Test token insert, validate, touch, delete lifecycle
  - [x] Test probe config upsert (insert then update)
  - [x] Test outlet event insert and list
- [x] [verify] `go test ./internal/db/...` passes (both DuckDB and SQLite tests)

---

## 2.2 First-Run Token Bootstrap

↳ depends on: 2.1

- [x] [code] Create `internal/db/bootstrap.go`:
  - [x] `EnsureDefaultToken(sqlite *SQLiteDB) (string, bool, error)`
    - [x] Checks if `auth_tokens` is empty
    - [x] If empty: generates token, inserts with label `"default"`, returns `(token, true, nil)` — `true` means "newly created"
    - [x] If not empty: returns `("", false, nil)`
- [x] [code] In `cmd/api/main.go`: call `EnsureDefaultToken` on startup
  - [x] If newly created: print token to stdout with clear formatting
- [ ] [verify] Delete SQLite DB, start API server, confirm token printed once
- [ ] [verify] Restart API server with existing DB, confirm token not printed again

---

## 2.3 HTTP Server Setup

↳ depends on: 2.1, 2.2

- [x] [code] Create `internal/api/server.go`:
  - [x] `Server` struct: DuckDB, SQLite, Apex client, Broadcaster, config
  - [x] `New(cfg *config.Config, duck *db.DuckDB, sqlite *db.SQLiteDB, apex apex.Client) *Server`
  - [x] `Run(ctx context.Context) error` — starts HTTP server, blocks until ctx cancelled
  - [x] Route registration (see 2.4 for handler implementations)
  - [x] Graceful shutdown: `http.Server.Shutdown(ctx)` on context cancellation
- [x] [code] Create `internal/api/middleware.go`:
  - [x] `RequestID` middleware — generates UUID, attaches to context and `X-Request-ID` header
  - [x] `Logger` middleware — structured log per request: method, path, status, duration_ms, request_id
  - [x] `Recover` middleware — catches panics, logs stack trace, returns 500
  - [x] `CORS` middleware — allows `http://localhost:5173` (Vite dev server) and same-origin
  - [x] `Auth` middleware — validates Bearer token via SQLite, updates `last_used` async
    - [x] Skip auth for `GET /api/stream` path (uses query param token instead)
  - [x] Middleware chain applied in order: RequestID → Logger → Recover → CORS → Auth → handler
- [x] [code] Create `internal/api/helpers.go`:
  - [x] `writeJSON(w, status int, v any)` — sets Content-Type, marshals, writes
  - [x] `writeError(w, status int, msg, code string)` — consistent error shape
  - [x] `readJSON(r, v any) error` — decodes request body with size limit (1MB)
  - [x] `queryParam(r, key, defaultVal string) string`
  - [x] `requireParam(r, key string) (string, error)` — returns 400 if missing
- [x] [code] Create `cmd/api/main.go`:
  - [x] Load config
  - [x] Set up slog JSON logger
  - [x] Open DuckDB (read-only)
  - [x] Open SQLite (read-write)
  - [x] Create Apex client
  - [x] Bootstrap default token
  - [x] Create and start API server
  - [x] Handle SIGTERM/SIGINT
- [x] [verify] `go build ./cmd/api` compiles
- [ ] [verify] `./api` starts and listens on port 8420
- [ ] [verify] `curl http://localhost:8420/` returns something (even 404 is fine at this stage)

---

## 2.4 API Handlers

↳ depends on: 2.3

### Probe Handlers

- [x] [code] Create `internal/api/probes.go`:
  - [x] `HandleProbeList(w, r)`:
    - [x] Call `db.CurrentProbeReadings(ctx)`
    - [x] For each probe: load `ProbeConfig` from SQLite (merge display_name, unit_override)
    - [x] Compute `status` field: compare value against `min_normal`/`max_normal`/`min_warning`/`max_warning`
    - [x] Write JSON response matching architecture spec
  - [x] `HandleProbeHistory(w, r)`:
    - [x] Extract `name` from URL path
    - [x] Parse `from`, `to`, `interval` query params
    - [x] Apply auto-interval selection if `interval` not provided (see architecture doc table)
    - [x] Call `db.ProbeHistory(ctx, name, from, to, interval)`
    - [x] Write JSON response
- [x] [test] `internal/api/probes_test.go`:
  - [x] Test list returns correctly shaped JSON
  - [x] Test status computation: normal, warning, critical, unknown
  - [x] Test history with explicit interval
  - [x] Test history with auto-interval selection
  - [x] Test 404 for unknown probe name

### Outlet Handlers

- [x] [code] Create `internal/api/outlets.go`:
  - [x] `HandleOutletList(w, r)`:
    - [x] Call `db.CurrentOutletStates(ctx)`
    - [x] Merge `OutletConfig` display names from SQLite
    - [x] Write JSON response
  - [x] `HandleOutletSet(w, r)`:
    - [x] Extract outlet `id` from URL path
    - [x] Read and validate request body `{ state: "ON"|"OFF"|"AUTO" }`
    - [x] Map user-facing state to Apex state: "AUTO" → "AON" (returns outlet to program control)
    - [x] Fetch current state from `status[0]` (for `from_state` in event log)
    - [x] Call `apex.SetOutlet(ctx, did, state)` — sends PUT to `/rest/status/outputs/<did>`
    - [x] On success: `sqlite.InsertOutletEvent(...)` with `initiated_by = "api"`
    - [x] Return updated outlet state
    - [x] On Apex error: return 502 with descriptive error
  - [x] `HandleOutletEvents(w, r)`:
    - [x] Extract outlet `id` from URL path (optional — empty means all outlets)
    - [x] Parse `limit` query param (default 50, max 200)
    - [x] Call `sqlite.ListOutletEvents(id, limit)`
    - [x] Write JSON response
- [x] [test] `internal/api/outlets_test.go`:
  - [x] Test list returns correct shape
  - [x] Test set with mock Apex — verify event logged
  - [x] Test set with Apex returning error — verify 502
  - [x] Test invalid state value returns 400
  - [x] Test events list with and without outlet_id filter

### System Handler

- [x] [code] Create `internal/api/system.go`:
  - [x] `HandleSystemStatus(w, r)`:
    - [x] Call `db.ControllerMeta(ctx)` from DuckDB
    - [x] Call `db.LastPollTime(ctx)` from DuckDB
    - [x] Get DuckDB file size via `os.Stat`
    - [x] Get SQLite file size via `os.Stat`
    - [x] Determine `poll_ok`: last poll time within 2× poll interval
    - [x] Write JSON response
- [x] [test] Test system handler returns correct shape and `poll_ok` logic

### Config Handlers

- [x] [code] Create `internal/api/config.go`:
  - [x] `HandleProbeConfigList(w, r)` — list all probe configs
  - [x] `HandleProbeConfigUpdate(w, r)` — upsert probe config by name
  - [x] `HandleOutletConfigList(w, r)` — list all outlet configs
  - [x] `HandleOutletConfigUpdate(w, r)` — upsert outlet config by id
- [x] [test] Test config CRUD roundtrip

### Alert Handlers

- [x] [code] Create `internal/api/alerts.go`:
  - [x] `HandleAlertList(w, r)` — list all alert rules
  - [x] `HandleAlertCreate(w, r)` — validate and insert rule
  - [x] `HandleAlertUpdate(w, r)` — update existing rule
  - [x] `HandleAlertDelete(w, r)` — delete rule by id
- [x] [test] Test alert CRUD: create, list, update, delete
- [x] [test] Test validation: invalid condition type, missing threshold, invalid severity

### Auth Handlers

- [x] [code] Create `internal/api/auth.go`:
  - [x] `HandleTokenList(w, r)` — list tokens (never return token value, only id/label/created/last_used)
  - [x] `HandleTokenCreate(w, r)` — create token with label, return token value once
  - [x] `HandleTokenDelete(w, r)` — delete by id
- [x] [test] Test token lifecycle via API

---

## 2.5 SSE Broadcaster

↳ depends on: 2.3

- [x] [code] Create `internal/api/sse.go`:
  - [x] `Event` struct: `Type string`, `Data any`
  - [x] `Broadcaster` struct: `sync.RWMutex`, `clients map[string]chan Event`
  - [x] `NewBroadcaster() *Broadcaster`
  - [x] `Subscribe(id string) <-chan Event` — creates buffered channel (buffer 5), registers client
  - [x] `Unsubscribe(id string)` — removes client, closes channel
  - [x] `Publish(e Event)` — sends to all clients; if channel full (slow client), skip (non-blocking send)
  - [x] `HandleStream(w, r)`:
    - [x] Validate token from `?token=` query param (same SQLite lookup as middleware)
    - [x] Set SSE headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`
    - [x] Subscribe to broadcaster with unique client ID (UUID)
    - [x] Defer unsubscribe
    - [x] Loop: select on client channel or `r.Context().Done()`
    - [x] Write events in SSE format: `event: <type>\ndata: <json>\n\n`
    - [x] Send heartbeat every 30s if no other events
- [x] [code] In `cmd/api/main.go`: create Broadcaster, pass to Server
- [x] [code] Integrate Broadcaster with Poller notification:
  - [x] After each successful poll, publish `probe_update` and `outlet_update` events
  - [x] **Option A (simple):** Poller has a notification channel; API server reads it
  - [x] **Option B (clean):** Use a shared in-process pub/sub (single process in Phase 2)
  - [x] Decision: since Poller and API are separate processes, SSE initially triggers on API's own polling of DuckDB every 10s, until IPC is needed
  - [x] Implement a background goroutine in the API server that polls DuckDB every 10s and publishes to Broadcaster — this is sufficient and removes the need for IPC
- [x] [test] `internal/api/sse_test.go`:
  - [x] Test Subscribe/Publish/Unsubscribe lifecycle
  - [x] Test slow client is skipped (non-blocking publish)
  - [x] Test broadcaster publish reaches multiple subscribers
- [ ] [verify] `curl -N -H "Authorization: Bearer <token>" http://localhost:8420/api/stream` shows events every 10s

---

## 2.6 Static Frontend Serving

- [ ] [code] In `internal/api/server.go`:
  - [ ] Add file server for `frontend/dist/` directory
  - [ ] Route: `GET /*` → serve static files
  - [ ] SPA fallback: if file not found, serve `index.html` (enables React Router client-side routing)
  - [ ] `SYMBIONT_FRONTEND_PATH` config var controls directory (default: `./frontend/dist`)
- [ ] [verify] After Phase 4, frontend will be served from here. For now, verify `/` returns 404 or empty gracefully.

---

## 2.7 API Integration Tests

↳ depends on: all handlers in 2.4

- [ ] [test] Create `internal/api/integration_test.go`:
  - [ ] Use `httptest.NewServer` with real (in-memory) DuckDB and SQLite instances
  - [ ] Seed test data into DuckDB
  - [ ] Seed token into SQLite
  - [ ] Test full request/response cycle for each endpoint
  - [ ] Test auth middleware rejects missing token (401)
  - [ ] Test auth middleware rejects wrong token (401)
  - [ ] Test auth middleware accepts valid token
- [ ] [verify] `go test ./internal/api/...` passes

---

## 2.8 API Verification (Manual)

↳ depends on: 2.7, Poller service running with real data

- [ ] [verify] `curl -s -H "Authorization: Bearer <token>" http://localhost:8420/api/probes | jq .`
  - [ ] Confirms probe names, values, and status fields
- [ ] [verify] `curl -s -H "Authorization: Bearer <token>" http://localhost:8420/api/probes/Temp/history?interval=1m | jq .`
  - [ ] Confirms bucketed data points in correct shape
- [ ] [verify] `curl -s -H "Authorization: Bearer <token>" http://localhost:8420/api/outlets | jq .`
  - [ ] Confirms outlet names and states
- [ ] [verify] `curl -s -X PUT -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"state":"OFF"}' http://localhost:8420/api/outlets/<id>`
  - [ ] Apex outlet physically toggles
  - [ ] Event logged in SQLite: `sqlite3 /var/lib/symbiont/app.db "SELECT * FROM outlet_event_log"`
- [ ] [verify] `curl -s -H "Authorization: Bearer <token>" http://localhost:8420/api/system | jq .`
  - [ ] `poll_ok` is `true`
  - [ ] Firmware and serial are populated
- [ ] [verify] SSE stream delivers events:
  - [ ] `curl -N "http://localhost:8420/api/stream?token=<token>"` — events appear every 10s

---

## 2.9 NixOS API Service

↳ depends on: 2.8 verified

- [ ] [config] Add `symbiont-api` systemd service to `flake.nix` (matching architecture doc)
- [ ] [verify] `sudo systemctl start symbiont-api` → active
- [ ] [verify] `sudo systemctl status symbiont-api` → healthy
- [ ] [verify] Both `symbiont-poller` and `symbiont-api` running simultaneously without conflicts
- [ ] [verify] API still responds after OS reboot (both services auto-start)

---

## Phase 2 Checklist Summary

- [ ] SQLite schema and all query functions
- [ ] First-run token bootstrap
- [ ] HTTP server with middleware stack
- [ ] All API handlers implemented and tested
- [ ] SSE broadcaster with background polling
- [ ] Manual verification of all endpoints against real Apex data
- [ ] Both systemd services running on NixOS

**Phase 2 is complete when:** Every endpoint in the API design returns correct data from curl, outlet control physically affects the Apex, and SSE events appear in real time.
