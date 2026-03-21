# Symbiont — Phase 6: Alerts, Notifications, and Polish
> Alert engine, ntfy.sh, data export, backup automation, retention

**Deliverable:** Symbiont is the complete primary interface for the tank. Alerts fire and notify. Data can be exported. Backups run automatically. Symbiont is strictly better than Fusion in daily use.

---

## 6.1 Alert Engine

↳ depends on: Phase 2 (SQLite schema with alert_rules), Phase 4 (SSE broadcaster running)

- [ ] [code] Create `internal/alerts/engine.go`:
  - [ ] `Engine` struct:
    - [ ] `sqlite *db.SQLiteDB`
    - [ ] `duck *db.DuckDB`
    - [ ] `notifier notify.Notifier`
    - [ ] `broadcaster *api.Broadcaster`
    - [ ] `state map[int64]AlertState` — in-memory active state per rule ID
    - [ ] `mu sync.Mutex`
  - [ ] `AlertState` struct:
    - [ ] `Active bool`
    - [ ] `FiredAt time.Time`
    - [ ] `ClearedAt time.Time`
    - [ ] `PeakValue float64`
    - [ ] `EventID int64` — SQLite `alert_events.id` for the current open event
    - [ ] `LastNotifiedAt time.Time` — for cooldown enforcement
  - [ ] `New(sqlite, duck, notifier, broadcaster) *Engine`
  - [ ] `Start(ctx context.Context)` — starts background goroutine, evaluates on 10s ticker
  - [ ] `Evaluate(ctx context.Context)` — loads latest probe values, evaluates all enabled rules
  - [ ] `isBreached(rule AlertRule, value float64) bool` — condition logic:
    - [ ] `above`: `value > threshold_high`
    - [ ] `below`: `value < threshold_low`
    - [ ] `outside_range`: `value < threshold_low || value > threshold_high`
  - [ ] `fire(ctx, rule, probe)`:
    - [ ] Set `state[rule.ID].Active = true`
    - [ ] Insert `alert_events` row in SQLite
    - [ ] Call `notifier.Send()` (respecting cooldown)
    - [ ] Publish `alert_fired` event to SSE broadcaster
  - [ ] `clear(ctx, rule)`:
    - [ ] Set `state[rule.ID].Active = false`
    - [ ] Update `alert_events.cleared_at` in SQLite
    - [ ] Publish `alert_cleared` event to SSE broadcaster (optional — for future UI badge updates)
  - [ ] `updatePeak(rule, value)`:
    - [ ] Update peak in memory
    - [ ] If cooldown expired, re-notify (prevents repeated alerts every 10s but re-alerts if still breached after cooldown)

- [ ] [code] Wire `Engine` into `cmd/api/main.go`:
  - [ ] Create engine after server is initialized
  - [ ] `go engine.Start(ctx)` — runs alongside HTTP server
  - [ ] Engine uses same DuckDB read-only connection as API server

- [ ] [test] Create `internal/alerts/engine_test.go`:
  - [ ] Test `isBreached` for all three condition types
  - [ ] Test `fire` is called when condition first breached (not on every subsequent eval)
  - [ ] Test `clear` is called when condition resolves
  - [ ] Test cooldown: `fire` not re-called within cooldown period
  - [ ] Test cooldown: `fire` IS called after cooldown period expires while still breached
  - [ ] Test engine handles missing probe gracefully (probe in rule doesn't exist in data)

- [ ] [verify] Create a test alert rule (e.g. temp above 60°F — always true)
- [ ] [verify] Alert fires within 10 seconds
- [ ] [verify] Alert does not re-fire on next eval (debounced)
- [ ] [verify] `SELECT * FROM alert_events` in SQLite shows fired event
- [ ] [verify] Adjust threshold so condition clears — `cleared_at` populated in DB

---

## 6.2 ntfy.sh Notification Delivery

- [ ] [code] Create `internal/notify/notifier.go`:
  - [ ] `Notifier` interface: `Send(ctx context.Context, n Notification) error`
  - [ ] `Notification` struct: `Title`, `Body`, `Priority`, `Tags []string`
  - [ ] `Priority` type: `"default"`, `"high"`, `"urgent"`
- [ ] [code] Create `internal/notify/ntfy.go`:
  - [ ] `NtfyNotifier` struct: `topicURL string`, `client *http.Client`
  - [ ] `NewNtfy(topicURL string) *NtfyNotifier`
  - [ ] `Send(ctx, n Notification) error`:
    - [ ] POST to `<topicURL>` with:
      - [ ] Body: `n.Body`
      - [ ] `Title` header: `n.Title`
      - [ ] `Priority` header: `n.Priority`
      - [ ] `Tags` header: comma-separated tags (e.g. `warning,thermometer`)
    - [ ] Retry once on transient failure (5xx, network error) with 5s delay
    - [ ] Return error on persistent failure
- [ ] [code] Create `internal/notify/multi.go`:
  - [ ] `MultiNotifier` struct: `[]Notifier`
  - [ ] `Send(ctx, n) error` — sends to all notifiers, collects errors, returns combined error if any fail
  - [ ] Enables adding webhook or other delivery types without changing engine
- [ ] [code] Create `internal/notify/noop.go`:
  - [ ] `NoopNotifier` — for use in tests where notification side effects aren't wanted
- [ ] [code] Load notification targets from SQLite `notification_targets` table on engine startup
  - [ ] Each enabled target with `type = "ntfy"` creates an `NtfyNotifier`
  - [ ] Pass `MultiNotifier` to alert engine
- [ ] [code] Add notification alert content:
  - [ ] Title: `"⚠️ pH Warning"` or `"🚨 Temperature Critical"` based on severity
  - [ ] Body: `"pH is 7.8 (threshold: below 8.0). Check your tank."`
  - [ ] Tags: severity, probe type
  - [ ] Priority: `"high"` for warning, `"urgent"` for critical
- [ ] [test] Create `internal/notify/ntfy_test.go`:
  - [ ] Mock ntfy server with `httptest`
  - [ ] Test correct headers are set
  - [ ] Test retry on 500
  - [ ] Test no retry on 400 (bad request — not transient)
- [ ] [verify] Configure real ntfy.sh topic URL in Settings UI
- [ ] [verify] Trigger a test notification from Settings page
- [ ] [verify] Phone receives notification via ntfy app

---

## 6.3 Add Notification Test Endpoint

- [ ] [code] Add `POST /api/notifications/test` to API server:
  - [ ] Sends a test notification to all enabled targets
  - [ ] Returns success/failure per target
- [ ] [code] Wire up in Settings UI `Notifications` tab (already has "Test" button from Phase 4)
- [ ] [verify] "Test" button in Settings sends real notification to phone

---

## 6.4 Alert Events API Endpoint

- [ ] [code] Add `GET /api/alerts/events` to API server (stubbed in Phase 2, now implemented):
  - [ ] Query params: `limit` (default 50), `rule_id` (optional filter), `active_only` (bool)
  - [ ] Joins `alert_events` with `alert_rules` for probe name and severity in response
  - [ ] Returns fired_at, cleared_at (null if still active), peak_value, probe_name, severity
- [ ] [code] Wire up in frontend Alerts page event log (Phase 4 UI was stubbed)
- [ ] [code] Add `alert_fired` and `alert_cleared` SSE event handling in frontend:
  - [ ] `alert_fired` → show toast notification in browser
  - [ ] Invalidate `['alerts', 'events']` query to refresh event log
- [ ] [verify] Active alerts appear in Alerts page event log
- [ ] [verify] Browser shows toast when alert fires

---

## 6.5 Data Export

- [ ] [code] Add `GET /api/probes/:name/export` to API server:
  - [ ] Query params: `from`, `to`, `format` (`csv` only for now)
  - [ ] Returns `Content-Disposition: attachment; filename="Temp-2025-03-20.csv"`
  - [ ] CSV format: `timestamp,value,unit`
  - [ ] Streams from DuckDB (avoid loading entire dataset into memory)
  - [ ] For large ranges: use DuckDB cursor / chunked reads
- [ ] [code] Add `GET /api/export` bulk export:
  - [ ] Downloads all probes for a date range as a ZIP with one CSV per probe
  - [ ] Use `archive/zip` in Go, stream to response writer
- [ ] [code] Add export buttons to frontend History page:
  - [ ] "Export CSV" button appears below chart for selected probe + time range
  - [ ] Calls export endpoint with current `from`/`to` range
  - [ ] Triggers browser download
- [ ] [verify] Export CSV for Temp probe over last 7 days
- [ ] [verify] Open CSV in spreadsheet app — columns and data look correct
- [ ] [verify] Bulk export ZIP contains one file per probe

---

## 6.6 Automated Backup

↳ depends on: `internal/db/sqlite_queries.go` backup job functions from Phase 2

- [ ] [code] Create `internal/backup/backup.go`:
  - [ ] `RunBackup(ctx context.Context, duck *db.DuckDB, sqlite *db.SQLiteDB, cfg BackupConfig) (*BackupResult, error)`
  - [ ] `BackupConfig`: `BackupPath string`, `Retain int`
  - [ ] Steps:
    1. Generate timestamp-based filenames: `telemetry-2025-03-20.db`, `app-2025-03-20.db`
    2. Copy DuckDB file: use DuckDB's `CHECKPOINT` then `os.Link` or `io.Copy` (DuckDB safe copy method)
    3. Copy SQLite file: use SQLite `.backup` API or `io.Copy` after `PRAGMA wal_checkpoint(FULL)`
    4. Verify copy sizes match source
    5. Prune old backups beyond `Retain` count
    6. Record result in SQLite `backup_jobs` table
  - [ ] Returns `BackupResult`: status, paths, sizes, error
  - [ ] `PruneOldBackups(backupPath string, retain int) error` — deletes oldest files beyond retain count

- [ ] [code] Add `POST /api/system/backup` handler (was stubbed in Phase 2):
  - [ ] Runs backup synchronously (or async with job ID — sync is simpler for now)
  - [ ] Returns backup result JSON

- [ ] [code] Add `GET /api/system/backups` handler:
  - [ ] Lists backup jobs from SQLite `backup_jobs` table
  - [ ] Used by Settings Backup tab

- [ ] [config] Add `symbiont-backup` systemd timer to `flake.nix`:
  - [ ] Timer: `OnCalendar = "daily"`, `Persistent = true`
  - [ ] Service: `ExecStart = symbiont system backup --quiet`
  - [ ] Service type: `oneshot`

- [ ] [test] Create `internal/backup/backup_test.go`:
  - [ ] Test backup creates files in correct location
  - [ ] Test pruning removes oldest files beyond retain count
  - [ ] Test backup job is recorded in SQLite
  - [ ] Test backup fails gracefully if disk is full (mock `os.Copy` failure)

- [ ] [verify] `symbiont system backup` creates files in `/var/lib/symbiont/backups/`
- [ ] [verify] Settings Backup tab shows last backup status
- [ ] [verify] Systemd timer runs daily (check with `systemctl list-timers`)
- [ ] [verify] After 2 runs: `ls /var/lib/symbiont/backups/` shows both files

---

## 6.7 Automated Data Retention

- [ ] [code] Complete `internal/db/cleanup.go` (stubbed in Phase 1):
  - [ ] `DeleteOldRows(ctx, duck *db.DuckDB, retentionDays int) (*CleanupResult, error)`
  - [ ] Deletes from all four DuckDB tables: `WHERE ts < NOW() - INTERVAL '<n> days'`
  - [ ] Returns row counts deleted per table
  - [ ] Runs `VACUUM` after deletion (DuckDB equivalent: `CHECKPOINT`)
  - [ ] Logs before/after database file sizes

- [ ] [code] Add `POST /api/system/cleanup` handler for manual trigger (admin use)

- [ ] [code] Add `symbiont system cleanup` CLI subcommand

- [ ] [config] Add `symbiont-cleanup` systemd timer to `flake.nix`:
  - [ ] Timer: `OnCalendar = "weekly"`, `Persistent = true`
  - [ ] Service: `ExecStart = symbiont system cleanup`

- [ ] [verify] `symbiont system cleanup` removes rows older than retention threshold
- [ ] [verify] DuckDB file size is stable/decreasing over long operation (no unbounded growth)

---

## 6.8 Outlet Event Log Improvements

- [ ] [code] Add `initiated_by` filter to `GET /api/outlets/events`:
  - [ ] Query param: `initiated_by=mcp` — useful for auditing AI actions
- [ ] [code] Frontend Outlets page: add `initiated_by` filter dropdown
- [ ] [code] Add pagination to outlet event log API (`page`, `page_size` params)
- [ ] [code] Frontend: "Load more" button on event log (or cursor-based pagination)
- [ ] [verify] Filter to `initiated_by=mcp` shows only AI-initiated changes

---

## 6.9 System Health Improvements

- [ ] [code] Add poller health tracking:
  - [ ] Poller writes a heartbeat file (e.g. `/var/lib/symbiont/poller.heartbeat`) every 30s with PID and timestamp
  - [ ] API server reads this file to determine true `poll_ok` (not just "last DuckDB row was recent")
  - [ ] If heartbeat is stale (>60s), `poll_ok = false` even if DuckDB has recent data
  - [ ] Note: Poller must NOT write to SQLite (architectural constraint — SQLite is API-server-only)
- [ ] [code] Add `GET /api/system/log` endpoint:
  - [ ] Returns last N structured log lines from poller and API (read from journald via exec, or from a log file)
  - [ ] Useful for debugging from the browser without SSH
- [ ] [code] Frontend Settings: add "System Log" tab showing recent log lines

---

## 6.10 Final Polish

- [ ] [code] Review all API error messages for clarity and consistency
- [ ] [code] Ensure all CLI `--json` output is valid JSON parseable by `jq`
- [ ] [code] Add `GET /api/healthz` unauthenticated health check endpoint (returns 200 if API is up)
  - [ ] Used by load balancers, uptime monitors, etc.
- [ ] [code] Review all frontend empty states — every page should have a useful empty state
- [ ] [code] Add `robots.txt` and `manifest.json` to frontend public assets
- [ ] [config] Review systemd hardening on all services — check for any overly permissive `ReadWritePaths`
- [ ] [verify] Run all Go tests: `go test ./...` — all pass
- [ ] [verify] Run frontend build: `npm run build` — no errors or warnings
- [ ] [verify] TypeScript: `npx tsc --noEmit` — no type errors
- [ ] [verify] End-to-end scenario: reboot NixOS mini PC → all services start → dashboard usable within 30s

---

## Phase 6 Checklist Summary

- [ ] Alert evaluation engine with debounce and cooldown
- [ ] ntfy.sh notification delivery with retry
- [ ] Alert events API endpoint and frontend UI integration
- [ ] Data export (CSV per probe, bulk ZIP)
- [ ] Automated daily backup with pruning
- [ ] Automated weekly retention cleanup
- [ ] Outlet event log improvements and pagination
- [ ] System health and `poll_ok` improvements
- [ ] All tests passing
- [ ] End-to-end reboot verification

**Phase 6 is complete when:** Symbiont sends a real notification to your phone when a parameter goes out of range, data can be exported, backups run automatically, and the system is fully self-managing.
