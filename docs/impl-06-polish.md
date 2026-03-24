# Symbiont — Phase 6: Alerts, Notifications, and Polish
> Alert engine, ntfy.sh, data export, backup automation, retention

**Deliverable:** Symbiont is the complete primary interface for the tank. Alerts fire and notify. Data can be exported. Backups run automatically. Symbiont is strictly better than Fusion in daily use.

---

## 6.1 Alert Engine

↳ depends on: Phase 2 (SQLite schema with alert_rules), Phase 4 (SSE broadcaster running)

- [x] [code] Create `internal/alerts/engine.go`:
  - [x] `Engine` struct:
    - [x] `sqlite *db.SQLiteDB`
    - [x] `duck *db.DuckDB`
    - [x] `notifier notify.Notifier`
    - [x] `broadcaster *api.Broadcaster`
    - [x] `state map[int64]AlertState` — in-memory active state per rule ID
    - [x] `mu sync.Mutex`
  - [x] `AlertState` struct:
    - [x] `Active bool`
    - [x] `FiredAt time.Time`
    - [x] `ClearedAt time.Time`
    - [x] `PeakValue float64`
    - [x] `EventID int64` — SQLite `alert_events.id` for the current open event
    - [x] `LastNotifiedAt time.Time` — for cooldown enforcement
  - [x] `New(sqlite, duck, notifier, broadcaster) *Engine`
  - [x] `Start(ctx context.Context)` — starts background goroutine, evaluates on 10s ticker
  - [x] `Evaluate(ctx context.Context)` — loads latest probe values, evaluates all enabled rules
  - [x] `isBreached(rule AlertRule, value float64) bool` — condition logic:
    - [x] `above`: `value > threshold_high`
    - [x] `below`: `value < threshold_low`
    - [x] `outside_range`: `value < threshold_low || value > threshold_high`
  - [x] `fire(ctx, rule, probe)`:
    - [x] Set `state[rule.ID].Active = true`
    - [x] Insert `alert_events` row in SQLite
    - [x] Call `notifier.Send()` (respecting cooldown)
    - [x] Publish `alert_fired` event to SSE broadcaster
  - [x] `clear(ctx, rule)`:
    - [x] Set `state[rule.ID].Active = false`
    - [x] Update `alert_events.cleared_at` in SQLite
    - [x] Publish `alert_cleared` event to SSE broadcaster (optional — for future UI badge updates)
  - [x] `updatePeak(rule, value)`:
    - [x] Update peak in memory
    - [x] If cooldown expired, re-notify (prevents repeated alerts every 10s but re-alerts if still breached after cooldown)

- [x] [code] Wire `Engine` into `cmd/api/main.go`:
  - [x] Create engine after server is initialized
  - [x] `go engine.Start(ctx)` — runs alongside HTTP server
  - [x] Engine uses same DuckDB read-only connection as API server

- [x] [test] Create `internal/alerts/engine_test.go`:
  - [x] Test `isBreached` for all three condition types
  - [x] Test `fire` is called when condition first breached (not on every subsequent eval)
  - [x] Test `clear` is called when condition resolves
  - [x] Test cooldown: `fire` not re-called within cooldown period
  - [x] Test cooldown: `fire` IS called after cooldown period expires while still breached
  - [x] Test engine handles missing probe gracefully (probe in rule doesn't exist in data)

- [ ] [verify] Create a test alert rule (e.g. temp above 60°F — always true)
- [ ] [verify] Alert fires within 10 seconds
- [ ] [verify] Alert does not re-fire on next eval (debounced)
- [ ] [verify] `SELECT * FROM alert_events` in SQLite shows fired event
- [ ] [verify] Adjust threshold so condition clears — `cleared_at` populated in DB

---

## 6.2 ntfy.sh Notification Delivery

- [x] [code] Create `internal/notify/notifier.go`:
  - [x] `Notifier` interface: `Send(ctx context.Context, n Notification) error`
  - [x] `Notification` struct: `Title`, `Body`, `Priority`, `Tags []string`
  - [x] `Priority` type: `"default"`, `"high"`, `"urgent"`
- [x] [code] Create `internal/notify/ntfy.go`:
  - [x] `NtfyNotifier` struct: `topicURL string`, `client *http.Client`
  - [x] `NewNtfy(topicURL string) *NtfyNotifier`
  - [x] `Send(ctx, n Notification) error`:
    - [x] POST to `<topicURL>` with:
      - [x] Body: `n.Body`
      - [x] `Title` header: `n.Title`
      - [x] `Priority` header: `n.Priority`
      - [x] `Tags` header: comma-separated tags (e.g. `warning,thermometer`)
    - [x] Retry once on transient failure (5xx, network error) with 5s delay
    - [x] Return error on persistent failure
- [x] [code] Create `internal/notify/multi.go`:
  - [x] `MultiNotifier` struct: `[]Notifier`
  - [x] `Send(ctx, n) error` — sends to all notifiers, collects errors, returns combined error if any fail
  - [x] Enables adding webhook or other delivery types without changing engine
- [x] [code] Create `internal/notify/noop.go`:
  - [x] `NoopNotifier` — for use in tests where notification side effects aren't wanted
- [x] [code] Load notification targets from SQLite `notification_targets` table on engine startup
  - [x] Each enabled target with `type = "ntfy"` creates an `NtfyNotifier`
  - [x] Pass `MultiNotifier` to alert engine
- [x] [code] Add notification alert content:
  - [x] Title: `"⚠️ pH Warning"` or `"🚨 Temperature Critical"` based on severity
  - [x] Body: `"pH is 7.8 (threshold: below 8.0). Check your tank."`
  - [x] Tags: severity, probe type
  - [x] Priority: `"high"` for warning, `"urgent"` for critical
- [x] [test] Create `internal/notify/ntfy_test.go`:
  - [x] Mock ntfy server with `httptest`
  - [x] Test correct headers are set
  - [x] Test retry on 500
  - [x] Test no retry on 400 (bad request — not transient)
- [ ] [verify] Configure real ntfy.sh topic URL in Settings UI
- [ ] [verify] Trigger a test notification from Settings page
- [ ] [verify] Phone receives notification via ntfy app

---

## 6.3 Add Notification Test Endpoint

- [x] [code] Add `POST /api/notifications/test` to API server:
  - [x] Sends a test notification to all enabled targets
  - [x] Returns success/failure per target
- [x] [code] Add notification targets CRUD API:
  - [x] `GET /api/notifications/targets`
  - [x] `POST /api/notifications/targets`
  - [x] `DELETE /api/notifications/targets/{id}`
- [ ] [code] Wire up in Settings UI `Notifications` tab (already has "Test" button from Phase 4)
- [ ] [verify] "Test" button in Settings sends real notification to phone

---

## 6.4 Alert Events API Endpoint

- [x] [code] Add `GET /api/alerts/events` to API server (stubbed in Phase 2, now implemented):
  - [x] Query params: `limit` (default 50), `rule_id` (optional filter), `active_only` (bool)
  - [x] Joins `alert_events` with `alert_rules` for probe name and severity in response
  - [x] Returns fired_at, cleared_at (null if still active), peak_value, probe_name, severity
- [x] [code] Wire up in frontend Alerts page event log (Phase 4 UI was stubbed)
- [x] [code] Add `alert_fired` and `alert_cleared` SSE event handling in frontend:
  - [x] `alert_fired` → show toast notification in browser
  - [x] Invalidate `['alerts', 'events']` query to refresh event log
- [ ] [verify] Active alerts appear in Alerts page event log
- [ ] [verify] Browser shows toast when alert fires

---

## 6.5 Data Export

- [x] [code] Add `GET /api/probes/:name/export` to API server:
  - [x] Query params: `from`, `to`
  - [x] Returns `Content-Disposition: attachment; filename="Temp-2025-03-20.csv"`
  - [x] CSV format: `timestamp,value`
  - [x] Streams from DuckDB (avoid loading entire dataset into memory)
- [x] [code] Add `GET /api/export` bulk export:
  - [x] Downloads all probes for a date range as a ZIP with one CSV per probe
  - [x] Use `archive/zip` in Go, stream to response writer
- [x] [code] Add export buttons to frontend History page:
  - [x] "Export CSV" button appears below chart for selected probe + time range
  - [x] Calls export endpoint with current `from`/`to` range
  - [x] Triggers browser download via `<a href>` with token query param
- [ ] [verify] Export CSV for Temp probe over last 7 days
- [ ] [verify] Open CSV in spreadsheet app — columns and data look correct
- [ ] [verify] Bulk export ZIP contains one file per probe

---

## 6.6 Automated Backup

↳ depends on: `internal/db/sqlite_queries.go` backup job functions from Phase 2

- [x] [code] Create `internal/backup/backup.go`:
  - [x] `Run(ctx, duck, sqlite, cfg BackupConfig) (*BackupResult, error)`
  - [x] `BackupConfig`: `BackupDir string`, `Retain int`
  - [x] Steps:
    1. Generate timestamp-based filenames: `telemetry-2025-03-20.db`, `app-2025-03-20.db`
    2. Copy DuckDB file: use DuckDB's `CHECKPOINT` then `io.Copy`
    3. Copy SQLite file: `PRAGMA wal_checkpoint(FULL)` then `io.Copy`
    4. Prune old backups beyond `Retain` count
    5. Record result in SQLite `backup_jobs` table (via API handler)
  - [x] Returns `BackupResult`: status, paths, sizes, error
  - [x] `PruneOldBackups(backupPath string, retain int) error` — deletes oldest files beyond retain count

- [x] [code] Add `POST /api/system/backup` handler:
  - [x] Runs backup synchronously
  - [x] Records job in SQLite
  - [x] Returns backup result JSON

- [x] [code] Add `GET /api/system/backups` handler:
  - [x] Lists backup jobs from SQLite `backup_jobs` table
  - [x] Used by Settings Backup tab

- [x] [code] Add `symbiont system backup` CLI command

- [ ] [config] Add `symbiont-backup` systemd timer to `flake.nix`:
  - [ ] Timer: `OnCalendar = "daily"`, `Persistent = true`
  - [ ] Service: `ExecStart = symbiont system backup --quiet`
  - [ ] Service type: `oneshot`

- [x] [test] Create `internal/backup/backup_test.go`:
  - [x] Test backup creates files in correct location
  - [x] Test pruning removes oldest files beyond retain count
  - [x] Test backup job is recorded in SQLite

- [ ] [verify] `symbiont system backup` creates files in `/var/lib/symbiont/backups/`
- [ ] [verify] Settings Backup tab shows last backup status
- [ ] [verify] Systemd timer runs daily (check with `systemctl list-timers`)
- [ ] [verify] After 2 runs: `ls /var/lib/symbiont/backups/` shows both files

---

## 6.7 Automated Data Retention

- [x] [code] `internal/db/cleanup.go` complete:
  - [x] `DeleteOldRows(ctx, retentionDays int) (*CleanupResult, error)`
  - [x] Deletes from all four DuckDB tables: `WHERE ts < NOW() - INTERVAL '<n> days'`
  - [x] Returns row counts deleted per table

- [x] [code] Add `POST /api/system/cleanup` handler for manual trigger (admin use)

- [x] [code] Add `symbiont system cleanup` CLI subcommand

- [ ] [config] Add `symbiont-cleanup` systemd timer to `flake.nix`:
  - [ ] Timer: `OnCalendar = "weekly"`, `Persistent = true`
  - [ ] Service: `ExecStart = symbiont system cleanup`

- [ ] [verify] `symbiont system cleanup` removes rows older than retention threshold
- [ ] [verify] DuckDB file size is stable/decreasing over long operation (no unbounded growth)

---

## 6.8 Outlet Event Log Improvements

- [x] [code] Add `initiated_by` filter to `GET /api/outlets/events`:
  - [x] Query param: `initiated_by=mcp` — useful for auditing AI actions
- [x] [code] Frontend Outlets page: add `initiated_by` filter dropdown
- [x] [code] Frontend: "Load more" button on event log (cursor-based pagination) — already existed
- [ ] [verify] Filter to `initiated_by=mcp` shows only AI-initiated changes

---

## 6.9 System Health Improvements

- [x] [code] Add poller health tracking:
  - [x] Poller writes a heartbeat file (e.g. `/var/lib/symbiont/poller.heartbeat`) after each poll with PID and timestamp
  - [x] API server reads this file to determine true `poll_ok` (not just "last DuckDB row was recent")
  - [x] If heartbeat is stale (>60s), `poll_ok = false` even if DuckDB has recent data
  - [x] Note: Poller must NOT write to SQLite (architectural constraint — SQLite is API-server-only)
- [ ] [code] Add `GET /api/system/log` endpoint:
  - [ ] Returns last N structured log lines from poller and API (read from journald via exec, or from a log file)
  - [ ] Useful for debugging from the browser without SSH
- [ ] [code] Frontend Settings: add "System Log" tab showing recent log lines

---

## 6.10 Final Polish

- [ ] [code] Review all API error messages for clarity and consistency
- [ ] [code] Ensure all CLI `--json` output is valid JSON parseable by `jq`
- [x] [code] Add `GET /api/healthz` unauthenticated health check endpoint (returns 200 if API is up)
  - [x] Used by load balancers, uptime monitors, etc.
- [ ] [code] Review all frontend empty states — every page should have a useful empty state
- [x] [code] Add `robots.txt` and `manifest.json` to frontend public assets
- [ ] [config] Review systemd hardening on all services — check for any overly permissive `ReadWritePaths`
- [x] [verify] Run all Go tests: `go test ./...` — all pass
- [x] [verify] Run frontend build: `npm run build` — no errors or warnings
- [x] [verify] TypeScript: `npx tsc --noEmit` — no type errors
- [ ] [verify] End-to-end scenario: reboot NixOS mini PC → all services start → dashboard usable within 30s

---

## Phase 6 Checklist Summary

- [x] Alert evaluation engine with debounce and cooldown
- [x] ntfy.sh notification delivery with retry
- [x] Alert events API endpoint and frontend UI integration
- [x] Data export (CSV per probe, bulk ZIP)
- [x] Automated daily backup with pruning
- [x] Automated weekly retention cleanup
- [x] Outlet event log improvements and pagination
- [x] System health and `poll_ok` improvements
- [x] All tests passing
- [ ] End-to-end reboot verification

**Phase 6 is complete when:** Symbiont sends a real notification to your phone when a parameter goes out of range, data can be exported, backups run automatically, and the system is fully self-managing.
