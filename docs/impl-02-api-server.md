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
- [ ] [code] In `cmd/api/main.go`: call `EnsureDefaultToken` on startup *(wired in 2.3)*
  - [ ] If newly created: print token to stdout with clear formatting
- [ ] [verify] Delete SQLite DB, start API server, confirm token printed once *(verified in 2.3)*
- [ ] [verify] Restart API server with existing DB, confirm token not printed again *(verified in 2.3)*

---

## 2.3 HTTP Server Setup

↳ depends on: 2.1, 2.2

- [ ] [code] Create `internal/api/server.go`:
  - [ ] `Server` struct: DuckDB, SQLite, Apex client, Broadcaster, config
  - [ ] `New(cfg *config.Config, duck *db.DuckDB, sqlite *db.SQLiteDB, apex apex.Client) *Server`
  - [ ] `Run(ctx context.Context) error` — starts HTTP server, blocks until ctx cancelled
  - [ ] Route registration (see 2.4 for handler implementations)
  - [ ] Graceful shutdown: `http.Server.Shutdown(ctx)` on context cancellation
- [ ] [code] Create `internal/api/middleware.go`:
  - [ ] `RequestID` middleware — generates UUID, attaches to context and `X-Request-ID` header
  - [ ] `Logger` middleware — structured log per request: method, path, status, duration_ms, request_id
  - [ ] `Recover` middleware — catches panics, logs stack trace, returns 500
  - [ ] `CORS` middleware — allows `http://localhost:5173` (Vite dev server) and same-origin
  - [ ] `Auth` middleware — validates Bearer token via SQLite, updates `last_used` async
    - [ ] Skip auth for `GET /api/stream` path (uses query param token instead)
  - [ ] Middleware chain applied in order: RequestID → Logger → Recover → CORS → Auth → handler
- [ ] [code] Create `internal/api/helpers.go`:
  - [ ] `writeJSON(w, status int, v any)` — sets Content-Type, marshals, writes
  - [ ] `writeError(w, status int, msg, code string)` — consistent error shape
  - [ ] `readJSON(r, v any) error` — decodes request body with size limit (1MB)
  - [ ] `queryParam(r, key, defaultVal string) string`
  - [ ] `requireParam(r, key string) (string, error)` — returns 400 if missing
- [ ] [code] Create `cmd/api/main.go`:
  - [ ] Load config
  - [ ] Set up slog JSON logger
  - [ ] Open DuckDB (read-only)
  - [ ] Open SQLite (read-write)
  - [ ] Create Apex client
  - [ ] Bootstrap default token
  - [ ] Create and start API server
  - [ ] Handle SIGTERM/SIGINT
- [ ] [verify] `go build ./cmd/api` compiles
- [ ] [verify] `./api` starts and listens on port 8420
- [ ] [verify] `curl http://localhost:8420/` returns something (even 404 is fine at this stage)

---

## 2.4 API Handlers

↳ depends on: 2.3

### Probe Handlers

- [ ] [code] Create `internal/api/probes.go`:
  - [ ] `HandleProbeList(w, r)`:
    - [ ] Call `db.CurrentProbeReadings(ctx)`
    - [ ] For each probe: load `ProbeConfig` from SQLite (merge display_name, unit_override)
    - [ ] Compute `status` field: compare value against `min_normal`/`max_normal`/`min_warning`/`max_warning`
    - [ ] Write JSON response matching architecture spec
  - [ ] `HandleProbeHistory(w, r)`:
    - [ ] Extract `name` from URL path
    - [ ] Parse `from`, `to`, `interval` query params
    - [ ] Apply auto-interval selection if `interval` not provided (see architecture doc table)
    - [ ] Call `db.ProbeHistory(ctx, name, from, to, interval)`
    - [ ] Write JSON response
- [ ] [test] `internal/api/probes_test.go`:
  - [ ] Test list returns correctly shaped JSON
  - [ ] Test status computation: normal, warning, critical, unknown
  - [ ] Test history with explicit interval
  - [ ] Test history with auto-interval selection
  - [ ] Test 404 for unknown probe name

### Outlet Handlers

- [ ] [code] Create `internal/api/outlets.go`:
  - [ ] `HandleOutletList(w, r)`:
    - [ ] Call `db.CurrentOutletStates(ctx)`
    - [ ] Merge `OutletConfig` display names from SQLite
    - [ ] Write JSON response
  - [ ] `HandleOutletSet(w, r)`:
    - [ ] Extract outlet `id` from URL path
    - [ ] Read and validate request body `{ state: "ON"|"OFF"|"AUTO" }`
    - [ ] Map user-facing state to Apex state: "AUTO" → "AON" (returns outlet to program control)
    - [ ] Fetch current state from `status[0]` (for `from_state` in event log)
    - [ ] Call `apex.SetOutlet(ctx, did, state)` — sends PUT to `/rest/status/outputs/<did>`
    - [ ] On success: `sqlite.InsertOutletEvent(...)` with `initiated_by = "api"`
    - [ ] Return updated outlet state
    - [ ] On Apex error: return 502 with descriptive error
  - [ ] `HandleOutletEvents(w, r)`:
    - [ ] Extract outlet `id` from URL path (optional — empty means all outlets)
    - [ ] Parse `limit` query param (default 50, max 200)
    - [ ] Call `sqlite.ListOutletEvents(id, limit)`
    - [ ] Write JSON response
- [ ] [test] `internal/api/outlets_test.go`:
  - [ ] Test list returns correct shape
  - [ ] Test set with mock Apex — verify event logged
  - [ ] Test set with Apex returning error — verify 502
  - [ ] Test invalid state value returns 400
  - [ ] Test events list with and without outlet_id filter

### System Handler

- [ ] [code] Create `internal/api/system.go`:
  - [ ] `HandleSystemStatus(w, r)`:
    - [ ] Call `db.ControllerMeta(ctx)` from DuckDB
    - [ ] Call `db.LastPollTime(ctx)` from DuckDB
    - [ ] Get DuckDB file size via `os.Stat`
    - [ ] Get SQLite file size via `os.Stat`
    - [ ] Determine `poll_ok`: last poll time within 2× poll interval
    - [ ] Write JSON response
- [ ] [test] Test system handler returns correct shape and `poll_ok` logic

### Config Handlers

- [ ] [code] Create `internal/api/config.go`:
  - [ ] `HandleProbeConfigList(w, r)` — list all probe configs
  - [ ] `HandleProbeConfigUpdate(w, r)` — upsert probe config by name
  - [ ] `HandleOutletConfigList(w, r)` — list all outlet configs
  - [ ] `HandleOutletConfigUpdate(w, r)` — upsert outlet config by id
- [ ] [test] Test config CRUD roundtrip

### Alert Handlers

- [ ] [code] Create `internal/api/alerts.go`:
  - [ ] `HandleAlertList(w, r)` — list all alert rules
  - [ ] `HandleAlertCreate(w, r)` — validate and insert rule
  - [ ] `HandleAlertUpdate(w, r)` — update existing rule
  - [ ] `HandleAlertDelete(w, r)` — delete rule by id
- [ ] [test] Test alert CRUD: create, list, update, delete
- [ ] [test] Test validation: invalid condition type, missing threshold, invalid severity

### Auth Handlers

- [ ] [code] Create `internal/api/auth.go`:
  - [ ] `HandleTokenList(w, r)` — list tokens (never return token value, only id/label/created/last_used)
  - [ ] `HandleTokenCreate(w, r)` — create token with label, return token value once
  - [ ] `HandleTokenDelete(w, r)` — delete by id
- [ ] [test] Test token lifecycle via API

---

## 2.5 SSE Broadcaster

↳ depends on: 2.3

- [ ] [code] Create `internal/api/sse.go`:
  - [ ] `Event` struct: `Type string`, `Data any`
  - [ ] `Broadcaster` struct: `sync.RWMutex`, `clients map[string]chan Event`
  - [ ] `NewBroadcaster() *Broadcaster`
  - [ ] `Subscribe(id string) <-chan Event` — creates buffered channel (buffer 5), registers client
  - [ ] `Unsubscribe(id string)` — removes client, closes channel
  - [ ] `Publish(e Event)` — sends to all clients; if channel full (slow client), skip (non-blocking send)
  - [ ] `HandleStream(w, r)`:
    - [ ] Validate token from `?token=` query param (same SQLite lookup as middleware)
    - [ ] Set SSE headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`
    - [ ] Subscribe to broadcaster with unique client ID (UUID)
    - [ ] Defer unsubscribe
    - [ ] Loop: select on client channel or `r.Context().Done()`
    - [ ] Write events in SSE format: `event: <type>\ndata: <json>\n\n`
    - [ ] Send heartbeat every 30s if no other events
- [ ] [code] In `cmd/api/main.go`: create Broadcaster, pass to Server
- [ ] [code] Integrate Broadcaster with Poller notification:
  - [ ] After each successful poll, publish `probe_update` and `outlet_update` events
  - [ ] **Option A (simple):** Poller has a notification channel; API server reads it
  - [ ] **Option B (clean):** Use a shared in-process pub/sub (single process in Phase 2)
  - [ ] Decision: since Poller and API are separate processes, SSE initially triggers on API's own polling of DuckDB every 10s, until IPC is needed
  - [ ] Implement a background goroutine in the API server that polls DuckDB every 10s and publishes to Broadcaster — this is sufficient and removes the need for IPC
- [ ] [test] `internal/api/sse_test.go`:
  - [ ] Test Subscribe/Publish/Unsubscribe lifecycle
  - [ ] Test slow client is skipped (non-blocking publish)
  - [ ] Test broadcaster publish reaches multiple subscribers
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
