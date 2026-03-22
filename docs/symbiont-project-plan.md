# Symbiont — Project Plan
> A local-first Neptune Apex dashboard and data platform

---

## Vision

Replace Apex Fusion with a self-hosted stack that collects, stores, and visualizes all your
aquarium data locally — with full outlet control — and no dependency on Neptune's cloud.

**Design principles:**
- Fully local, no cloud dependency
- Runs on the existing NixOS mini PC
- Go backend, TypeScript/React frontend
- DuckDB for time-series telemetry (embedded, no server process)
- SQLite for relational app state and configuration
- Clean REST/JSON API decoupled from Apex's quirky auth
- AI-first platform — MCP server and CLI as first-class integration surfaces
- Sleek, minimal UI with depth available on demand

---

## What We Know About the Apex (AOS 5+)

### Local API — Status

The Apex runs a local web server. AOS 5+ exposes a proper REST API:

| Endpoint | Method | Description |
|---|---|---|
| `/rest/login` | POST | Authenticate, get session token |
| `/rest/status` | GET | Full JSON: probes, outputs, controller info |
| `/rest/outlets` | GET | Outlet states only |
| `/rest/outlets/{id}` | PUT | Control a single outlet |

Legacy endpoints still work on newer firmware for read-only use:
- `GET /cgi-bin/status.xml` — XML status dump
- `GET /cgi-bin/status.json` — JSON status dump

### Auth

Newer AOS uses session-cookie auth (not basic auth):
1. POST credentials to `/rest/login`
2. Server sets a `connect.sid` session cookie
3. All subsequent requests include that cookie
4. **Session expires** — the poller must detect 401s and re-authenticate

We'll need to capture the exact login request/response from browser DevTools against
your unit to confirm the payload shape and cookie behavior. This is the first
reverse-engineering task.

### Data Available from `/rest/status`

**Inputs (probes):**
- Temperature
- pH
- ORP
- Salinity/conductivity
- Any PM module probes (Ca, Alk, Mg via Trident if present)

**Outputs:**
- Per-outlet: name, state (ON/OFF/AON/AOF), type, intensity

**Controller:**
- Firmware version, serial number
- `power_failed` / `power_restored` timestamps
- System time (NTP-synced)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        NixOS Mini PC                         │
│                                                              │
│  ┌──────────────┐     ┌──────────────────────────────────┐  │
│  │   Poller     │────▶│           DuckDB                 │  │
│  │   (Go)       │     │       symbiont.db                │  │
│  │              │     │                                  │  │
│  │ Polls Apex   │     │  Tables:                         │  │
│  │ every 10s    │     │  - probe_readings                │  │
│  │ Writes rows  │     │  - outlet_states                 │  │
│  └──────────────┘     │  - power_events                  │  │
│                       │  - controller_meta               │  │
│  ┌──────────────┐     └──────────────────────────────────┘  │
│  │  API Server  │──────────────▲                            │
│  │  (Go/net     │              │ reads                      │
│  │   http)      │                                           │
│  │              │     ┌──────────────────────────────────┐  │
│  │  /api/probes │     │           SQLite                 │  │
│  │  /api/outlets│◀───▶│       symbiont-app.db            │  │
│  │  /api/system │     │                                  │  │
│  │  /api/config │     │  Tables:                         │  │
│  │  /api/alerts │     │  - alert_rules                   │  │
│  │  /api/control│     │  - probe_config                  │  │
│  └──────────────┘     │  - outlet_config                 │  │
│         │             │  - auth_tokens                   │  │
│         │             │  - outlet_event_log              │  │
│         │             │  - notification_targets          │  │
│         │             │  - backup_jobs                   │  │
│         ▼             └──────────────────────────────────┘  │
│  ┌──────────────┐                                           │
│  │   Frontend   │     ┌──────────────────────────────────┐  │
│  │  (Vite +     │     │       Neptune Apex               │  │
│  │   React)     │     │     (local network)              │  │
│  │              │     └──────────────────────────────────┘  │
│  │  Served by   │                    ▲                      │
│  │  API or      │                    │ outlet control       │
│  │  nginx       │     ┌──────────────┴───────────────────┐  │
│  └──────────────┘     │         API Server               │  │
│                       │   proxies all Apex commands      │  │
│  ┌──────────────┐     └──────────────────────────────────┘  │
│  │  MCP Server  │                                           │
│  │  (Go)        │                                           │
│  │              │                                           │
│  │  Wraps REST  │                                           │
│  │  API for AI  │                                           │
│  │  assistants  │                                           │
│  └──────────────┘                                           │
│                                                              │
│  ┌──────────────┐                                           │
│  │  CLI         │                                           │
│  │  (Go)        │                                           │
│  │              │                                           │
│  │  JSON output │                                           │
│  │  mode built  │                                           │
│  │  in          │                                           │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

**Go throughout the backend.** Single binaries per service, low memory footprint, stable
for long-running processes, clean goroutine model for the poller loop. Trivial NixOS
deployment. No GIL, no interpreter overhead, no dependency hell.

**REST/JSON, not gRPC/protobuf.** All communication between services is HTTP/JSON.
The browser can talk natively to the API, the CLI and MCP server are just HTTP clients,
and everything is trivially debuggable with curl. No proto files to maintain.

**Single writer to DuckDB.** DuckDB allows multiple readers but one writer at a time.
The Poller is the sole writer. The API server is read-only against the DuckDB. Outlet
control commands go directly from the API to the Apex.

**DuckDB vs SQLite — clear division of responsibility.**
- DuckDB is the tank's memory — append-only time-series telemetry, write-heavy, columnar
- SQLite is the app's brain — relational config, user-facing state, rarely written
- Raw SQL queries in both, no ORM

**Poller ≠ API server.** Separate Go binaries running as separate systemd services.
A frontend reload or API bug can never interrupt data collection.

**API proxies all Apex communication.** The frontend and MCP server never talk to the
Apex directly. All outlet control goes through the API, which holds the Apex session
and forwards commands. Apex auth logic lives in one place.

**MCP and CLI are API clients.** Nothing is duplicated. Both wrap the REST API — the
MCP server exposes tools that call API endpoints, the CLI does the same with human-
friendly and JSON output modes.

**Dark mode is the default.** Not a toggle, not an afterthought. The primary color scheme
is dark. A light mode can be added later.

---

## Database Design

### DuckDB — Time-Series Telemetry

```sql
-- Probe readings (temperature, pH, ORP, salinity, etc.)
CREATE TABLE probe_readings (
    ts          TIMESTAMPTZ NOT NULL,
    probe_name  VARCHAR     NOT NULL,
    probe_type  VARCHAR     NOT NULL,  -- 'temp', 'pH', 'ORP', 'salinity', etc.
    value       DOUBLE      NOT NULL,
    unit        VARCHAR,
    PRIMARY KEY (ts, probe_name)
);

-- Outlet / output states
CREATE TABLE outlet_states (
    ts          TIMESTAMPTZ NOT NULL,
    outlet_id   VARCHAR     NOT NULL,
    outlet_name VARCHAR     NOT NULL,
    state       VARCHAR     NOT NULL,  -- 'ON', 'OFF', 'AON', 'AOF' (Apex-reported states)
    watts       DOUBLE,
    amps        DOUBLE,
    PRIMARY KEY (ts, outlet_id)
);

-- Power events from the controller
CREATE TABLE power_events (
    ts          TIMESTAMPTZ NOT NULL,
    event_type  VARCHAR     NOT NULL,  -- 'power_failed', 'power_restored'
    PRIMARY KEY (ts, event_type)
);

-- Controller metadata snapshots
CREATE TABLE controller_meta (
    ts              TIMESTAMPTZ NOT NULL,
    serial          VARCHAR,
    firmware        VARCHAR,
    hardware        VARCHAR,
    PRIMARY KEY (ts)
);
```

**Indexing:** DuckDB's columnar format makes range scans on `ts` fast by default.
No extra indexes needed for typical dashboard queries.

**Retention:** Systemd timer runs a weekly cleanup job deleting rows older than a
configurable threshold. 1 year of 10-second data ≈ 3M rows per probe — very
manageable for DuckDB.

**Backup:** Systemd timer runs a nightly `COPY` or file-level backup of the DuckDB
file to a configurable backup directory.

### SQLite — App State and Configuration

```sql
-- API authentication tokens
CREATE TABLE auth_tokens (
    id          INTEGER PRIMARY KEY,
    token       TEXT    NOT NULL UNIQUE,
    label       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used   DATETIME
);

-- Per-probe display configuration
CREATE TABLE probe_config (
    probe_name      TEXT PRIMARY KEY,
    display_name    TEXT,
    unit_override   TEXT,
    display_order   INTEGER,
    min_normal      REAL,
    max_normal      REAL,
    min_warning     REAL,
    max_warning     REAL
);

-- Per-outlet display configuration
CREATE TABLE outlet_config (
    outlet_id       TEXT PRIMARY KEY,
    display_name    TEXT,
    display_order   INTEGER,
    icon            TEXT
);

-- Alert rules
CREATE TABLE alert_rules (
    id              INTEGER PRIMARY KEY,
    probe_name      TEXT    NOT NULL,
    condition       TEXT    NOT NULL,  -- 'above', 'below', 'outside_range'
    threshold_low   REAL,
    threshold_high  REAL,
    severity        TEXT    NOT NULL,  -- 'warning', 'critical'
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Notification delivery targets
CREATE TABLE notification_targets (
    id          INTEGER PRIMARY KEY,
    type        TEXT    NOT NULL,  -- 'ntfy', 'webhook', etc.
    config      TEXT    NOT NULL,  -- JSON blob (URL, topic, etc.)
    enabled     INTEGER NOT NULL DEFAULT 1
);

-- Outlet event log (human and AI-initiated changes)
CREATE TABLE outlet_event_log (
    id          INTEGER PRIMARY KEY,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    outlet_id   TEXT     NOT NULL,
    outlet_name TEXT,
    from_state  TEXT,
    to_state    TEXT     NOT NULL,
    initiated_by TEXT    NOT NULL  -- 'ui', 'cli', 'mcp', 'api'
);

-- Backup job metadata
CREATE TABLE backup_jobs (
    id          INTEGER PRIMARY KEY,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status      TEXT     NOT NULL,  -- 'success', 'failed'
    path        TEXT,
    error       TEXT,
    size_bytes  INTEGER
);
```

---

## Project Structure

```
symbiont/
├── cmd/
│   ├── poller/
│   │   └── main.go             # Poller binary entrypoint
│   ├── api/
│   │   └── main.go             # API server binary entrypoint
│   ├── mcp/
│   │   └── main.go             # MCP server binary entrypoint
│   └── symbiont/
│       └── main.go             # CLI binary entrypoint
│
├── internal/
│   ├── apex/
│   │   ├── client.go           # Apex HTTP client (auth, session mgmt)
│   │   ├── models.go           # Go structs for Apex API responses
│   │   └── parser.go           # Normalize Apex response → internal models
│   ├── db/
│   │   ├── duckdb.go           # DuckDB connection and write functions
│   │   ├── sqlite.go           # SQLite connection and query functions
│   │   ├── schema.go           # Schema creation and migrations
│   │   └── queries.go          # Named query functions
│   ├── poller/
│   │   └── poller.go           # Polling loop (goroutines, ticker)
│   ├── api/
│   │   ├── server.go           # HTTP server setup, middleware, routing
│   │   ├── auth.go             # Token auth middleware
│   │   ├── probes.go           # GET /api/probes, /api/probes/{name}/history
│   │   ├── outlets.go          # GET /api/outlets, PUT /api/outlets/{id}
│   │   ├── system.go           # GET /api/system
│   │   ├── config.go           # GET/PUT /api/config (probe and outlet config)
│   │   └── alerts.go           # CRUD /api/alerts
│   ├── mcp/
│   │   └── tools.go            # MCP tool definitions wrapping API calls
│   ├── cli/
│   │   ├── probes.go           # symbiont probes [current|history]
│   │   ├── outlets.go          # symbiont outlets [list|set]
│   │   └── system.go           # symbiont system [status|backup]
│   ├── alerts/
│   │   └── engine.go           # Alert evaluation and notification dispatch
│   ├── notify/
│   │   └── ntfy.go             # ntfy.sh delivery implementation
│   └── config/
│       └── config.go           # App config (env vars, .env file)
│
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── api/
│   │   │   └── client.ts       # Typed fetch wrapper
│   │   ├── components/
│   │   │   ├── ProbeCard.tsx   # Current value + Tremor sparkline
│   │   │   ├── ProbeChart.tsx  # uPlot time-series chart wrapper
│   │   │   ├── OutletCard.tsx  # State badge + toggle control
│   │   │   ├── AlertBadge.tsx  # Threshold status indicator
│   │   │   └── PowerBadge.tsx  # Last power event indicator
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx   # Minimal overview — probe cards, outlet states
│   │   │   ├── History.tsx     # uPlot deep-dive, multi-probe, time range picker
│   │   │   ├── Outlets.tsx     # Outlet management and event log
│   │   │   ├── Alerts.tsx      # Alert rule configuration
│   │   │   └── Settings.tsx    # Config, auth tokens, notifications, backup
│   │   └── hooks/
│   │       ├── useProbes.ts
│   │       ├── useOutlets.ts
│   │       └── useSSE.ts       # Server-Sent Events for real-time updates
│   └── public/
│
├── flake.nix                   # NixOS dev environment and service definitions
└── README.md
```

---

## Go Dependencies

```
# Core
net/http                        # stdlib HTTP server — no framework needed
github.com/marcboeker/go-duckdb # DuckDB driver
modernc.org/sqlite              # SQLite driver (pure Go, no cgo)
github.com/joho/godotenv        # .env file loading

# API / serialization
encoding/json                   # stdlib

# CLI
github.com/spf13/cobra          # CLI framework with subcommands and JSON output

# MCP
github.com/mark3labs/mcp-go     # MCP server implementation

# Scheduling
time.Ticker                     # stdlib — sufficient for the poller loop

# Testing
net/http/httptest               # stdlib
```

---

## Frontend Dependencies

```json
{
  "dependencies": {
    "react": "^18",
    "react-dom": "^18",
    "uplot": "^1",
    "@tremor/react": "^3",
    "@tanstack/react-query": "^5",
    "lucide-react": "latest"
  },
  "devDependencies": {
    "typescript": "^5",
    "vite": "^5",
    "@types/react": "^18",
    "tailwindcss": "^3"
  }
}
```

**shadcn/ui** components added as needed via CLI — not a package dependency.

**Component hierarchy:**
- shadcn/ui — base primitives (buttons, dialogs, inputs, tables)
- Tremor — dashboard stat cards, sparklines, badge components
- uPlot — all time-series charting (wrapped in a reusable React component)

---

## API Design

### Authentication

All requests require an `Authorization: Bearer <token>` header. Tokens are generated
on first run and stored in SQLite. Rotation is manual via CLI or Settings page.

### Read Endpoints

```
GET /api/probes
→ { probes: [{ name, type, value, unit, ts, status }] }
  status: 'normal' | 'warning' | 'critical' based on alert rule thresholds

GET /api/probes/{name}/history?from=ISO&to=ISO&interval=1m
→ { probe, data: [{ ts, value }] }
  interval bucketed via DuckDB epoch math

GET /api/outlets
→ { outlets: [{ id, name, state, xstatus, watts, amps }] }

GET /api/system
→ { serial, firmware, last_poll_ts, poll_ok, db_size_bytes }

GET /api/alerts
→ { rules: [{ id, probe_name, condition, thresholds, severity, enabled }] }

GET /api/outlets/{id}/events?limit=50
→ { events: [{ ts, from_state, to_state, initiated_by }] }
```

### Write Endpoints

```
PUT /api/outlets/{id}
Body: { "state": "ON" | "OFF" | "AUTO" }
→ Sends command to Apex, logs to outlet_event_log, returns updated outlet state
→ ON/OFF use REST API; AUTO uses legacy CGI endpoint (state=0)

POST /api/alerts
PUT /api/alerts/{id}
DELETE /api/alerts/{id}
→ Alert rule management

PUT /api/config/probes/{name}
PUT /api/config/outlets/{id}
→ Display config (friendly names, ordering, units)

POST /api/auth/tokens
DELETE /api/auth/tokens/{id}
→ Token management
```

### Streaming

```
GET /api/stream   (Server-Sent Events)
→ Pushes probe + outlet updates every 10s to the frontend
  so the dashboard auto-refreshes without frontend polling
```

---

## MCP Server

The MCP server is a separate binary that wraps the REST API, exposing tools for
AI assistants (Claude, Gemini, etc.). All tools are thin clients over the API —
no direct DB access.

```
Tools:
  get_current_parameters      → calls GET /api/probes
  get_probe_history           → calls GET /api/probes/{name}/history
  get_outlet_states           → calls GET /api/outlets
  control_outlet              → calls PUT /api/outlets/{id}
  get_system_status           → calls GET /api/system
  get_alert_rules             → calls GET /api/alerts
  get_outlet_event_log        → calls GET /api/outlets/{id}/events
  summarize_tank_health       → calls multiple endpoints, returns synthesized summary
```

The token is configured in the MCP server's environment — not passed by the AI client.

**Example interactions enabled:**
- "Why did my pH drop last night?" — queries history, correlates with outlet events
- "Turn off the skimmer for an hour" — outlet control with natural language
- "Are all my parameters within range?" — health summary
- "What changed in my tank yesterday?" — outlet event log + parameter deltas

---

## CLI

Single binary `symbiont` with subcommands. All commands default to human-readable
output. `--json` flag available on every command for scripting and agent use.

```
symbiont probes current [--json]
symbiont probes history <name> [--from ISO] [--to ISO] [--interval 1m] [--json]
symbiont outlets list [--json]
symbiont outlets set <id> <ON|OFF|AUTO> [--json]
symbiont system status [--json]
symbiont system backup [--path /path/to/backup]
symbiont alerts list [--json]
symbiont auth tokens list
symbiont auth tokens create --label "claude-desktop"
symbiont auth tokens revoke <id>
```

---

## Auth

**Token-based, single shared secret model.** No user accounts, no sessions, no JWTs.

- One or more tokens stored in SQLite `auth_tokens` table
- Generated as cryptographically random 32-byte hex strings
- Required as `Authorization: Bearer <token>` on every API request
- First token auto-generated on first run, printed to stdout once
- Additional tokens created via CLI or Settings UI
- Each token has an optional label (e.g. "claude-desktop", "phone")
- `last_used` timestamp updated on each request for visibility
- Remote access via Tailscale — no ports exposed to the internet

---

## Alerting

Alert rules are stored in SQLite and evaluated by a background goroutine in the
API server process on each poller tick (via the SSE stream or internal channel).

**Rule types:**
- Value above threshold
- Value below threshold
- Value outside range (min/max)

**Severity levels:** warning, critical

**Delivery:** ntfy.sh (self-hosted or cloud). Config stored in SQLite
`notification_targets`. Additional delivery types (webhook, etc.) can be added
as new implementations of a `Notifier` interface.

**Alert state tracking:** debounced — only fires once when condition is first
breached, not on every poll. Re-arms when condition clears.

---

## Backup Strategy

**DuckDB:** Nightly systemd timer copies the DuckDB file to a configurable backup
directory. Keeps last N backups (configurable). Status recorded in SQLite
`backup_jobs` table and visible in the Settings UI.

**SQLite:** Included in the same nightly backup job. Both files backed up together
as a consistent snapshot.

**Recovery:** CLI command `symbiont system restore --from /path/to/backup` stops
services, replaces DB files, restarts.

---

## NixOS Integration

```nix
# flake.nix — systemd services

services.symbiont-poller = {
  description = "Symbiont Poller";
  wantedBy = [ "multi-user.target" ];
  after = [ "network.target" ];
  serviceConfig = {
    ExecStart = "${symbiont-poller}/bin/poller";
    Restart = "always";
    RestartSec = "5s";
    EnvironmentFile = "/etc/symbiont/env";
  };
};

services.symbiont-api = {
  description = "Symbiont API";
  wantedBy = [ "multi-user.target" ];
  after = [ "network.target" "symbiont-poller.service" ];
  serviceConfig = {
    ExecStart = "${symbiont-api}/bin/api --host 0.0.0.0 --port 8420";
    Restart = "always";
    EnvironmentFile = "/etc/symbiont/env";
  };
};

services.symbiont-mcp = {
  description = "Symbiont MCP Server";
  wantedBy = [ "multi-user.target" ];
  after = [ "symbiont-api.service" ];
  serviceConfig = {
    ExecStart = "${symbiont-mcp}/bin/mcp";
    Restart = "always";
    EnvironmentFile = "/etc/symbiont/env";
  };
};

# Nightly backup timer
systemd.timers.symbiont-backup = {
  wantedBy = [ "timers.target" ];
  timerConfig.OnCalendar = "daily";
};

# Weekly retention cleanup timer
systemd.timers.symbiont-cleanup = {
  wantedBy = [ "timers.target" ];
  timerConfig.OnCalendar = "weekly";
};
```

---

## Phased Roadmap

### Phase 1 — Data Collection
**Goal:** Poller running on NixOS, writing to DuckDB, verifiable via CLI

- [ ] Capture Apex login flow from browser DevTools (auth reverse-engineering)
- [ ] Build `apex/client.go` — login, session refresh on 401, GET status
- [ ] Define Go structs for Apex JSON response
- [ ] Set up DuckDB schema
- [ ] Build polling loop (goroutines + time.Ticker, 10s interval)
- [ ] Wire up as a systemd service via flake.nix

**Deliverable:** Poller binary runs cleanly; DuckDB file accumulates rows.

### Phase 2 — API Server
**Goal:** Clean REST API queryable from curl

- [ ] Go HTTP server with probe, outlet, and system endpoints
- [ ] SQLite schema and raw query functions
- [ ] Token auth middleware
- [ ] History endpoint with DuckDB time-bucketing
- [ ] Outlet control endpoint (proxies to Apex REST, logs to SQLite)
- [ ] SSE endpoint for real-time push
- [ ] CORS configured for local frontend dev

**Deliverable:** Full API documented and testable via curl.

### Phase 3 — CLI
**Goal:** Full CLI with JSON output mode

- [ ] Cobra CLI with all subcommands
- [ ] Human-readable and `--json` output modes on every command
- [ ] Token management commands
- [ ] Backup and restore commands

**Deliverable:** `symbiont` binary usable as a standalone tool and scripting surface.

### Phase 4 — Frontend MVP
**Goal:** Usable dashboard replacing daily Fusion use

- [ ] Vite + React + TypeScript + Tailwind + shadcn/ui scaffold
- [ ] Dark mode as default theme
- [ ] Dashboard page: Tremor probe stat cards, outlet state badges
- [ ] Outlets page: outlet cards with ON/OFF/AUTO toggle, event log
- [ ] History page: uPlot charts, multi-probe overlay, time range picker
- [ ] Alerts page: rule configuration UI
- [ ] Settings page: probe/outlet config, token management, notifications, backup status
- [ ] SSE integration for real-time updates
- [ ] Mobile-responsive layout

**Deliverable:** Full local dashboard accessible in browser.

### Phase 5 — MCP Server and AI Layer
**Goal:** Symbiont queryable and controllable by AI assistants

- [ ] MCP server binary with full tool surface
- [ ] All tools as thin API wrappers
- [ ] Tested against Claude Desktop and Claude Code
- [ ] `summarize_tank_health` composite tool

**Deliverable:** Claude can query and control the tank via MCP.

### Phase 6 — Alerts, Notifications, and Polish
**Goal:** Features that make it genuinely better than Fusion

- [ ] Alert evaluation engine with debounce
- [ ] ntfy.sh notification delivery
- [ ] Data export (CSV download for any probe range)
- [ ] Outlet event log UI improvements
- [ ] Backup automation and Settings UI integration
- [ ] Retention policy cleanup job

**Deliverable:** Symbiont is the primary and complete tank management interface.

### Phase 7 — Visual Layout Builder
**Goal:** Live schematic of the physical system

- [ ] React Flow canvas with edit and view modes
- [ ] Draggable probe and outlet nodes
- [ ] Layout config persisted as JSON in SQLite
- [ ] Live data overlaid on static layout
- [ ] Color-coded node status based on alert thresholds
- [ ] Click-through to history drawer and outlet control

**Deliverable:** Interactive tank diagram with live parameter overlay.

---

## Reverse Engineering Plan

The first step before writing a line of application code:

1. **Open Chrome DevTools** → Network tab → filter XHR/Fetch
2. **Browse to Apex local IP** and log in
3. **Capture:** the login POST (URL, body shape, response cookies)
4. **Capture:** a status GET (URL, headers sent including cookie, response JSON structure)
5. **Capture:** an outlet toggle PUT (URL, body shape, response)
6. **Capture:** a 401 flow if possible (let cookie expire or try without it)

This gives us the exact API contract to build `apex/client.go` against.
The community Go client and the Home Assistant Python integration
(`itchannel/apex-ha`) are useful cross-references but may be on slightly
different firmware — your DevTools capture is ground truth.

---

## Open Questions / Risks

| Question | Notes |
|---|---|
| Exact AOS 5+ login endpoint and payload | Resolve via DevTools capture |
| Session token lifetime | May be hours; need to test expiry behavior |
| Rate limits on local polling | Community reports 5-10s is safe; test with your unit |
| Outlet control payload shape | Needs verification — varies between firmware versions |
| Trident data availability | Check if Ca/Alk/Mg appear in `/rest/status` |
| WAV / Vortech xstatus | Extra field on wireless outputs; parse carefully |
| DuckDB Go driver stability | go-duckdb is CGO-based; test on NixOS specifically |
| Historical data backfill | Investigate whether Apex exposes any stored history for initial import |
