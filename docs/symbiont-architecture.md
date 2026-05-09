# Symbiont — Technical Architecture

> Last updated: May 2026

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Service Architecture](#2-service-architecture)
3. [Data Architecture](#3-data-architecture)
4. [Apex Client](#4-apex-client)
5. [Poller Service](#5-poller-service)
6. [API Server](#6-api-server)
7. [CLI](#7-cli)
8. [MCP Server](#8-mcp-server)
9. [Alert Engine](#9-alert-engine)
10. [Frontend Architecture](#10-frontend-architecture)
11. [Authentication](#11-authentication)
12. [Error Handling and Resilience](#12-error-handling-and-resilience)
13. [Data Flows](#13-data-flows)
14. [NixOS Deployment](#14-nixos-deployment)
15. [Configuration Reference](#15-configuration-reference)
16. [Performance Considerations](#16-performance-considerations)
17. [Security Considerations](#17-security-considerations)

---

## 1. System Overview

Symbiont is a local-first Neptune Apex dashboard and data platform. It replaces Apex Fusion with a self-hosted stack that owns the full data pipeline: collection, storage, API, and visualization. No cloud dependency. No Neptune account required after initial setup.

### Core Principles

**Single source of truth per concern.** DuckDB owns all time-series telemetry. SQLite owns all application state. The Apex owns outlet control authority. Nothing is duplicated across stores.

**Separation of data collection from API serving.** The Poller and API Server are independent binaries running as separate systemd services. A crash or restart in one cannot affect the other. The Poller writes; the API reads. Never the reverse.

**The API is the integration boundary.** The frontend, CLI, and MCP server are all HTTP clients of the same REST API. No component has privileged access to the database directly except the Poller (write) and API Server (read). This is enforced by convention and process isolation, not a network boundary.

**AI-first design.** The MCP server and CLI are first-class surfaces designed for agent and LLM consumption. JSON output modes, clean tool schemas, and composable query patterns are not afterthoughts.

### Technology Stack

| Layer | Technology | Rationale |
|---|---|---|
| Poller | Go | Long-running stability, low memory, goroutines |
| API Server | Go + net/http | Single binary, stdlib-first, no framework overhead |
| CLI | Go + Cobra | Same binary ecosystem, JSON output, scriptable |
| MCP Server | Go + mcp-go | Thin wrapper over API, same language |
| Time-series DB | DuckDB | Columnar, embedded, fast range scans, no server |
| App state DB | SQLite | Relational, embedded, raw SQL, no ORM |
| Frontend | React + TypeScript + Vite | Typed, fast builds, ecosystem |
| Charts | uPlot | Canvas-based, handles millions of points |
| Data fetching | TanStack Query | Cache, background refresh, SSE integration |
| Styling | Tailwind CSS | Utility-first, dark mode by default |
| Icons | lucide-react | Consistent icon set throughout |

---

## 2. Service Architecture

Symbiont runs as four separate binaries. Each is a self-contained Go binary deployed as a systemd service on NixOS.

```
┌─────────────────────────────────────────────────────────────────┐
│                         NixOS Host                               │
│                                                                  │
│   ┌─────────────────┐        ┌────────────────────────────────┐ │
│   │  symbiont-poller│        │         DuckDB                 │ │
│   │                 │──────▶ │    /var/lib/symbiont/           │ │
│   │  goroutine loop │  write │    telemetry.db                │ │
│   │  10s ticker     │        └────────────┬───────────────────┘ │
│   │  Apex session   │                     │ read                  │
│   └─────────────────┘                     │                       │
│                                           ▼                       │
│   ┌─────────────────┐        ┌────────────────────────────────┐ │
│   │  symbiont-api   │◀──────▶│         SQLite                 │ │
│   │                 │  r/w   │    /var/lib/symbiont/           │ │
│   │  :8420          │        │    app.db                      │ │
│   │  REST + SSE     │        └────────────────────────────────┘ │
│   │  Token auth     │                                            │
│   └────────┬────────┘                                            │
│            │                  ┌────────────────────────────────┐ │
│            │ HTTP/JSON        │       Neptune Apex             │ │
│            │ :8420            │    (local network)             │ │
│            │                  └────────────────────────────────┘ │
│   ┌────────┴────────┐                    ▲                       │
│   │  symbiont-mcp   │                    │ outlet control        │
│   │                 │                    │ proxied through API   │
│   │  MCP protocol   │                    │                       │
│   │  30+ tools      │                                            │
│   └─────────────────┘                                            │
│                                                                  │
│   ┌─────────────────┐                                            │
│   │  symbiont (CLI) │                                            │
│   │                 │                                            │
│   │  Cobra commands │                                            │
│   │  JSON output    │                                            │
│   └─────────────────┘                                            │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    Frontend (static)                     │   │
│   │   Served by symbiont-api on same host                    │   │
│   │   Vite build → /var/lib/symbiont/frontend/              │   │
│   └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Inter-Service Communication

There is no service mesh, no gRPC, no message bus. Communication patterns are:

- **Poller → DuckDB:** Direct library calls via `go-duckdb`. No network hop.
- **API Server → DuckDB:** Direct library calls, read-only connection.
- **API Server → SQLite:** Direct library calls via `modernc.org/sqlite`.
- **API Server → Apex:** HTTP/JSON via `net/http`. Session cookie managed by API server.
- **MCP Server → API:** HTTP/JSON to `localhost:8420`. Bearer token from env.
- **CLI → API:** HTTP/JSON to `localhost:8420`. Bearer token from env or config file.
- **Frontend → API:** HTTP/JSON and SSE to `localhost:8420` (or via Tailscale).

### Go Module Layout

```
module github.com/kjaebker/symbiont

go 1.23

require (
    github.com/marcboeker/go-duckdb v1.x
    modernc.org/sqlite v1.x
    github.com/spf13/cobra v1.x
    github.com/mark3labs/mcp-go v0.x
    github.com/joho/godotenv v1.x
)
```

All four binaries live in `cmd/`. All shared logic lives in `internal/`. No public packages — this is not a library.

---

## 3. Data Architecture

### Database Responsibilities

| Store | Owns | Write Pattern | Read Pattern |
|---|---|---|---|
| DuckDB | All time-series telemetry | Append-only, every 10s, single writer | Columnar range scans, dashboards, history |
| SQLite | App config, state, auth | Occasional, user-driven | Point lookups, small result sets |

These two databases are never joined. The API layer combines data from both when needed (e.g., probe current value from DuckDB + display config from SQLite), but at the application level, not via SQL.

### DuckDB Schema

```sql
-- Primary telemetry table
-- One row per probe per poll cycle
CREATE TABLE probe_readings (
    ts          TIMESTAMPTZ NOT NULL,
    probe_name  VARCHAR     NOT NULL,
    probe_type  VARCHAR     NOT NULL,
    value       DOUBLE      NOT NULL,
    unit        VARCHAR,
    PRIMARY KEY (ts, probe_name)
);

-- Outlet state snapshots
-- Written on every poll cycle (not only on change)
-- Enables accurate watt-hour calculations over time
CREATE TABLE outlet_states (
    ts          TIMESTAMPTZ NOT NULL,
    outlet_id   VARCHAR     NOT NULL,
    outlet_name VARCHAR     NOT NULL,
    state       VARCHAR     NOT NULL,  -- 'ON' | 'OFF' | 'AON' | 'AOF' (Apex-reported states)
    watts       DOUBLE,
    amps        DOUBLE,
    PRIMARY KEY (ts, outlet_id)
);

-- Power loss / restore events
-- Deduplicated by (ts, event_type) primary key
CREATE TABLE power_events (
    ts          TIMESTAMPTZ NOT NULL,
    event_type  VARCHAR     NOT NULL,  -- 'power_failed' | 'power_restored'
    PRIMARY KEY (ts, event_type)
);

-- Controller metadata snapshots
CREATE TABLE controller_meta (
    ts          TIMESTAMPTZ NOT NULL,
    serial      VARCHAR,
    firmware    VARCHAR,
    hardware    VARCHAR,
    PRIMARY KEY (ts)
);
```

### SQLite Schema

SQLite holds all application state. The schema is versioned via a `schema_versions` table and evolved through numbered migrations. Current version: 3.

Key tables:

```sql
-- Auth tokens
CREATE TABLE auth_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    token       TEXT     NOT NULL UNIQUE,
    label       TEXT,
    scope       TEXT     NOT NULL DEFAULT 'admin',  -- 'read' | 'write' | 'control' | 'admin'
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used   DATETIME
);

-- Physical devices (pumps, skimmers, heaters, etc.)
CREATE TABLE devices (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    device_type TEXT,
    description TEXT,
    brand       TEXT,
    model       TEXT,
    notes       TEXT,
    image_path  TEXT,
    outlet_id   TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Per-probe display and threshold configuration
CREATE TABLE probe_config (
    probe_name      TEXT PRIMARY KEY,
    display_name    TEXT,
    unit_override   TEXT,
    min_normal      REAL,
    max_normal      REAL,
    min_warning     REAL,
    max_warning     REAL,
    device_id       INTEGER REFERENCES devices(id),
    input_category  TEXT NOT NULL DEFAULT 'probe',
    on_label        TEXT,
    off_label       TEXT,
    ok_value        REAL,
    is_binary       INTEGER NOT NULL DEFAULT 0,
    hidden          INTEGER NOT NULL DEFAULT 0
);

-- Per-outlet display configuration
CREATE TABLE outlet_config (
    outlet_id   TEXT PRIMARY KEY,
    display_name TEXT,
    icon        TEXT
);

-- Device-outlet associations (many-to-many)
CREATE TABLE device_outlets (
    device_id   INTEGER NOT NULL REFERENCES devices(id),
    outlet_id   TEXT    NOT NULL,
    label       TEXT,
    color       TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (device_id, outlet_id)
);

-- Alert rules
CREATE TABLE alert_rules (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    probe_name      TEXT     NOT NULL,
    condition       TEXT     NOT NULL CHECK(condition IN ('above','below','outside_range')),
    threshold_low   REAL,
    threshold_high  REAL,
    severity        TEXT     NOT NULL CHECK(severity IN ('warning','critical')),
    cooldown_minutes INTEGER NOT NULL DEFAULT 30,
    enabled         INTEGER  NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Alert firing history
CREATE TABLE alert_events (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    rule_id     INTEGER  NOT NULL REFERENCES alert_rules(id),
    fired_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cleared_at  DATETIME,
    peak_value  REAL,
    notified    INTEGER  NOT NULL DEFAULT 0
);

-- Notification delivery targets
CREATE TABLE notification_targets (
    id      INTEGER  PRIMARY KEY AUTOINCREMENT,
    type    TEXT     NOT NULL,
    config  TEXT     NOT NULL,  -- JSON: {url, topic, priority, ...}
    label   TEXT,
    enabled INTEGER  NOT NULL DEFAULT 1
);

-- Backup job records
CREATE TABLE backup_jobs (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status      TEXT     NOT NULL CHECK(status IN ('success','failed')),
    path        TEXT,
    size_bytes  INTEGER,
    error       TEXT
);

-- Customizable dashboard items (probe, outlet, device, separator, feed_mode, measurement)
CREATE TABLE dashboard_items (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    item_type    TEXT     NOT NULL,
    reference_id TEXT,
    label        TEXT,
    sort_order   INTEGER  NOT NULL DEFAULT 0,
    display_mode TEXT     NOT NULL DEFAULT 'normal'
);

-- Water chemistry measurement parameters (Alkalinity, Calcium, Magnesium, etc.)
CREATE TABLE measurement_parameters (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    name           TEXT     NOT NULL UNIQUE,
    canonical_unit TEXT     NOT NULL,
    sort_order     INTEGER  NOT NULL DEFAULT 999
);

-- Manual water chemistry measurements
CREATE TABLE measurements (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    measured_at     DATETIME NOT NULL,
    parameter_id    INTEGER  NOT NULL REFERENCES measurement_parameters(id),
    value           REAL     NOT NULL,
    notes           TEXT,
    source          TEXT     NOT NULL DEFAULT 'manual',
    test_kit_ref    TEXT,
    raw_value       REAL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Livestock inventory (fish, coral, invertebrates, other)
CREATE TABLE livestock (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    species     TEXT,
    type        TEXT     NOT NULL CHECK(type IN ('fish','coral','invertebrate','other')),
    quantity    INTEGER  NOT NULL DEFAULT 1,
    status      TEXT     NOT NULL CHECK(status IN ('healthy','sick','quarantine','deceased')),
    date_added  DATE,
    notes       TEXT,
    image_path  TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Livestock health observations
CREATE TABLE livestock_observations (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    livestock_id INTEGER  NOT NULL REFERENCES livestock(id),
    ts           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status       TEXT,
    note         TEXT,
    image_path   TEXT
);

-- Tank physical profile (display tank + sump dimensions)
CREATE TABLE tank_profile (
    section         TEXT PRIMARY KEY,  -- 'display' or 'sump'
    shape           TEXT,
    length_in       REAL,
    width_in        REAL,
    height_in       REAL,
    diameter_in     REAL,
    net_volume_gal  REAL,
    notes           TEXT,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Journal entries (manual notes + auto-logged system events)
CREATE TABLE journal_entries (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type        TEXT     NOT NULL DEFAULT 'note',
    title       TEXT,
    body        TEXT,
    source      TEXT     NOT NULL DEFAULT 'user',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- AI agent settings
CREATE TABLE agent_settings (
    id                  INTEGER  PRIMARY KEY DEFAULT 1,
    tone                TEXT     NOT NULL DEFAULT 'analytical',
    dosing_product_line TEXT     NOT NULL DEFAULT 'none',
    net_volume_gallons  REAL,
    custom_guardrails   TEXT,
    enabled_skills      TEXT     NOT NULL DEFAULT '[]',
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- System event bus (audit log + journal auto-logging source)
CREATE TABLE events (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    ts           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind         TEXT     NOT NULL,
    initiated_by TEXT     NOT NULL,  -- indexed column (v3 migration)
    payload      TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Dosing products catalog
CREATE TABLE dosing_products (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    brand       TEXT     NOT NULL,
    name        TEXT     NOT NULL,
    type        TEXT     NOT NULL,
    unit        TEXT     NOT NULL DEFAULT 'mL',
    notes       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Dosing schedules
CREATE TABLE dosing_schedules (
    id                      INTEGER  PRIMARY KEY AUTOINCREMENT,
    product_id              INTEGER  NOT NULL REFERENCES dosing_products(id),
    amount                  REAL     NOT NULL,
    frequency               TEXT     NOT NULL,
    interval_days           INTEGER,
    day_of_week             INTEGER,
    enabled                 INTEGER  NOT NULL DEFAULT 1,
    last_completed_at       DATETIME,
    next_due_at             DATETIME,
    follow_up_parameter_id  INTEGER  REFERENCES measurement_parameters(id),
    follow_up_days          INTEGER,
    notes                   TEXT,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Dosing log
CREATE TABLE dosing_logs (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER  REFERENCES dosing_schedules(id),
    product_id  INTEGER  NOT NULL REFERENCES dosing_products(id),
    amount      REAL     NOT NULL,
    dosed_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes       TEXT,
    source      TEXT     NOT NULL DEFAULT 'manual',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Maintenance task definitions
CREATE TABLE maintenance_tasks (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL,
    description     TEXT,
    frequency       TEXT     NOT NULL,
    interval_days   INTEGER,
    day_of_week     INTEGER,
    enabled         INTEGER  NOT NULL DEFAULT 1,
    last_completed_at DATETIME,
    next_due_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Maintenance completion log
CREATE TABLE maintenance_logs (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER  NOT NULL REFERENCES maintenance_tasks(id),
    completed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes        TEXT,
    source       TEXT     NOT NULL DEFAULT 'manual',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Outlet programs and their audit history (intentional append-only audit log)
CREATE TABLE outlet_programs ( ... );
CREATE TABLE outlet_program_history ( ... );
```

---

## 4. Apex Client

The Apex client (`internal/apex/client.go`) is the only component that communicates with the Neptune Apex hardware. It is used by both the Poller (for status reads) and the API Server (for outlet control and feed mode). Each process holds its own client instance with its own session.

### Session Lifecycle

The Apex uses session-cookie auth. The client:
1. POSTs credentials to `/rest/login`
2. Stores the `connect.sid` session cookie
3. On any 401 response, automatically re-authenticates and retries the request once

The session mutex (`sync.Mutex`) prevents concurrent re-auth attempts.

---

## 5. Poller Service

The Poller is a long-running Go binary whose sole job is polling the Apex every 10 seconds and appending rows to DuckDB. It has no HTTP server, no config API, and no external interface.

On each tick:
1. `apex.Client.Status()` — fetches all probes and outlet states
2. `duckdb.WritePollCycle()` — writes inside a single transaction:
   - INSERT INTO probe_readings
   - INSERT INTO outlet_states
   - INSERT INTO power_events (conditional)
   - INSERT INTO controller_meta (conditional)

### Failure Behavior

- On Apex unreachable: log error, skip cycle, retry next tick. No crash.
- On DuckDB write failure: transaction rolls back, entire cycle skipped.
- On 401 from Apex: client re-authenticates automatically. Transparent to the Poller.
- On SIGTERM: graceful shutdown via context cancellation.

---

## 6. API Server

The API Server is a Go HTTP server (`cmd/api/main.go`) serving the REST API, SSE stream, and static frontend files. It reads from DuckDB (read-only) and reads/writes SQLite.

### Routing

The route table is large (~80 routes). Key groupings:

```
# Core telemetry
GET  /api/probes
GET  /api/probes/{name}/history
GET  /api/outlets
GET  /api/outlets/{id}/history
PUT  /api/outlets/{id}          [requires: control scope]
GET  /api/feed
PUT  /api/feed                  [requires: control scope]

# System
GET  /api/system
GET  /api/system/log
POST /api/system/backup         [requires: admin scope]
POST /api/system/cleanup        [requires: admin scope]
GET  /api/system/backups        [requires: admin scope]

# Config
GET  /api/config/probes
PUT  /api/config/probes/{name}  [requires: admin scope]
GET  /api/config/outlets
PUT  /api/config/outlets/{id}   [requires: admin scope]

# Devices
GET/POST     /api/devices
GET/PUT/DEL  /api/devices/{id}
PUT          /api/devices/{id}/probes
PUT          /api/devices/{id}/outlets
POST/DEL     /api/devices/{id}/image

# Dashboard
GET  /api/dashboard
PUT  /api/dashboard
POST /api/dashboard
DEL  /api/dashboard/{id}

# Alerts
GET/POST     /api/alerts
GET/PUT/DEL  /api/alerts/{id}
GET          /api/alerts/events

# Notifications
GET/POST     /api/notifications/targets
DEL          /api/notifications/targets/{id}
POST         /api/notifications/test

# Measurements (manual water chemistry)
GET          /api/measurements/parameters
GET          /api/measurements/kits
GET/POST     /api/measurements
PUT/DEL      /api/measurements/{id}

# Livestock
GET/POST     /api/livestock
GET/PUT/DEL  /api/livestock/{id}
POST/DEL     /api/livestock/{id}/image
GET/POST     /api/livestock/{id}/observations
POST/DEL     /api/livestock/{id}/observations/{obs_id}/image

# Tank profile
GET          /api/tank/profile
PUT          /api/tank/profile/display  [requires: admin scope]
PUT          /api/tank/profile/sump     [requires: admin scope]

# Events / Journal
GET  /api/events
GET  /api/events/stats
GET  /api/daily-prompt
POST /api/daily-prompt/respond
GET/POST     /api/journal
GET/PUT/DEL  /api/journal/{id}
GET          /api/journal/templates

# Dosing
GET/POST     /api/dosing/products
PUT/DEL      /api/dosing/products/{id}
GET/POST     /api/dosing/schedules
PUT/DEL      /api/dosing/schedules/{id}
POST         /api/dosing/schedules/{id}/log
GET          /api/dosing/logs

# Maintenance
GET/POST     /api/maintenance/tasks
PUT/DEL      /api/maintenance/tasks/{id}
POST         /api/maintenance/tasks/{id}/complete
GET          /api/maintenance/tasks/{id}/logs

# Due items (unified dosing + maintenance queue)
GET  /api/tasks/due

# Agent / AI
GET  /api/agent/settings
PUT  /api/agent/settings        [requires: admin scope]
GET  /api/agent/context
GET  /api/agent/skills
GET  /api/agent/skills/{name}/body

# MCP HTTP transport
POST/GET/DEL /api/mcp

# Auth tokens
GET/POST     /api/tokens        [requires: admin scope]
PATCH/DEL    /api/tokens/{id}   [requires: admin scope]

# Export
GET  /api/probes/{name}/export
GET  /api/export

# SSE
GET  /api/stream

# Static
GET  /images/{path}
GET  /*                          → SPA handler
```

### Middleware Stack

```
Request
  │
  ▼
RequestID (generate/attach trace ID)
  │
  ▼
Logger (structured log per request)
  │
  ▼
Recover (panic → 500, never crash)
  │
  ▼
CORS (allow local frontend dev origin)
  │
  ▼
Auth (validate Bearer token, attach scope to context)
  │
  ▼
Handler (optionally wrapped with s.admin() or s.control() scope check)
```

### Scope Enforcement

Token scope is attached to the request context by the `Auth` middleware. Route-level scope requirements use handler decorator wrappers:

```go
s.admin(handler)    // requires 'admin' scope
s.control(handler)  // requires 'control' or 'admin' scope
// no wrapper = any valid token (read, write, control, or admin)
```

Scope hierarchy: `read < write < control < admin`

- **read** — GET-only access (except `/api/mcp`)
- **write** — read + data management (journal, measurements, livestock, dosing, maintenance, dashboard, devices, alerts) — cannot control live hardware
- **control** — write + outlet control + feed mode
- **admin** — control + token management + config + backup + agent settings

### SSE Broadcaster

The SSE endpoint (`GET /api/stream`) runs two parallel feeds to clients:
- A 10-second DuckDB poller pushing probe + outlet snapshots
- A subscription to the internal event bus for real-time system events

---

## 7. CLI

The CLI binary (`symbiont`) is a Cobra-based command-line tool. It is a pure HTTP client of the API server. It never accesses the databases directly.

### Command Tree

```
symbiont
├── probes
│   ├── current               List current probe values
│   └── history <name>        Time-series history for one probe
│
├── outlets
│   ├── list                  List all outlet states
│   └── set <id> <state>      Control an outlet
│
├── alerts
│   ├── list                  List alert rules
│   ├── create                Create an alert rule
│   ├── update <id>           Update a rule
│   ├── delete <id>           Delete a rule
│   └── events                Recent alert firings
│
├── measurements
│   ├── list                  List water chemistry measurements
│   └── add                   Add a manual measurement
│
├── dosing
│   ├── products              List dosing products
│   ├── schedules             List dosing schedules
│   └── log                   Log a manual dose
│
├── livestock
│   ├── list                  List livestock inventory
│   └── add                   Add a livestock item
│
├── journal
│   ├── list                  List journal entries
│   └── add                   Add a journal entry
│
├── system
│   ├── status                Controller + poller health
│   └── backup                Trigger a manual backup
│
├── agent
│   ├── context               Get AI agent context bundle
│   └── skills                List available skills
│
└── auth
    ├── tokens list
    ├── tokens create --label <label> --scope <scope>
    └── tokens revoke <id>
```

Global flags: `--json`, `--api-url`, `--token`

---

## 8. MCP Server

The MCP Server exposes Symbiont's data and control capabilities to AI assistants via the Model Context Protocol. It communicates with the API Server over localhost.

### Transport Modes

The MCP server supports two transports:
- **stdio** — launched as a subprocess by Claude Desktop or Claude Code (local)
- **HTTP/SSE** — exposed at `/api/mcp` for claude.ai remote connections via Tailscale

### Tool Surface (30+ tools)

Organized into logical groups:

| Group | Tools |
|---|---|
| Probes | `get_current_parameters`, `get_probe_history`, `list_probe_configs`, `update_probe_config` |
| Outlets | `get_outlet_states`, `control_outlet`, `get_outlet_event_log` |
| Feed | `get_feed_mode`, `set_feed_mode` |
| System | `get_system_status`, `get_system_log` |
| Alerts | `get_alert_rules`, `create_alert_rule`, `update_alert_rule`, `delete_alert_rule`, `get_alert_events` |
| Measurements | `get_measurements`, `add_measurement`, `delete_measurement`, `get_measurement_parameters` |
| Livestock | `get_livestock`, `add_livestock`, `update_livestock`, `add_livestock_observation`, `get_livestock_observations`, `get_livestock_image` |
| Tank | `get_tank_profile`, `summarize_tank_health`, `get_agent_context` |
| Journal | `get_journal_entries`, `add_journal_entry`, `get_journal_templates` |
| Dosing | `get_dosing_products`, `get_dosing_schedule`, `create_dosing_schedule`, `get_dosing_history`, `log_dose` |
| Maintenance | `list_maintenance_tasks`, `create_maintenance_task`, `complete_maintenance_task`, `get_due_tasks` |
| Devices | `get_devices` |
| Skills | `list_skills`, `get_skill` |

All tools are thin HTTP clients over the REST API. No direct database access.

### Claude Desktop Configuration

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

### Remote MCP (claude.ai)

When accessed via Tailscale, `POST /api/mcp` serves the MCP HTTP+SSE transport, enabling claude.ai to connect without a local binary. See `docs/deployment-remote-mcp.md` for setup.

---

## 9. Alert Engine

The Alert Engine runs as a background goroutine inside the API Server process. It evaluates alert rules against the latest probe readings on every SSE broadcast cycle.

### Alert Evaluation

- Reads enabled alert rules from SQLite
- Compares latest probe values against each rule's thresholds
- On fire: inserts `alert_events` row, publishes notification
- On clear: updates `cleared_at`
- Each rule has a `cooldown_minutes` field to prevent notification spam

### Notification Interface

```go
type Notifier interface {
    Send(ctx context.Context, n Notification) error
}
```

The `ntfy.go` implementation POSTs to an ntfy.sh topic URL. Additional targets are configured in `notification_targets` SQLite table.

---

## 10. Frontend Architecture

### Page Structure

```
App
├── Layout
│   ├── Sidebar (nav)
│   └── TopBar (system status, last poll indicator)
│
├── /              → Dashboard (customizable via dashboard_items)
│   └── Probe, outlet, device, separator, feed mode, measurement cards
│
├── /history       → History (uPlot charts, multi-probe, time range picker)
│
├── /outlets       → Outlets (outlet cards, event log)
│
├── /chemistry     → Chemistry (measurements, dosing schedules, due items)
│
├── /livestock     → Livestock (fish/coral/invert inventory, observations)
│
├── /journal       → Journal (event log + manual entries + daily prompts)
│
├── /alerts        → Alerts (rule configuration, firing history)
│
└── /settings      → Settings (tabbed)
    ├── Dashboard   → drag-and-drop dashboard layout customization
    ├── Probes      → display names, units, thresholds, categories
    ├── Outlets     → display names, icons
    ├── Devices     → device management (create, edit, image upload)
    ├── Tank        → tank profile (dimensions, volumes)
    ├── Tokens      → API token management (create, scope, revoke)
    ├── Notifications → ntfy.sh setup and test
    ├── Backup      → last backup status, manual trigger
    ├── Agent       → AI agent settings (tone, product line, skills)
    └── System      → system log viewer
```

### State Management

No global state store. State lives at two levels:

**Server state** — TanStack Query with typed query key factory (`qk`):
```typescript
useQuery({ queryKey: qk.probes.all(), queryFn: api.probes.list })
```

**SSE invalidation** — the SSE hook invalidates query caches on server events:
```typescript
es.addEventListener('probe_update', () =>
  queryClient.invalidateQueries({ queryKey: qk.probes.all() })
)
```

**Local UI state** — React `useState` for form inputs, modal open/close, selected time ranges.

### Design System

The "Abyssal Laboratory" theme. Key rules:
- Surfaces use layered hierarchy (`surface` → `surface-container-*`) for depth
- No 1px solid borders — boundaries via background color shifts or ghost outlines
- No divider lines — spacing and surface shifts separate items
- Colors: Primary cyan (`#3adffa`), Secondary green (`#6dfe9c`), Tertiary coral (`#ff8796`)
- All transitions: 300ms ease-in-out (`transition-fluid`)
- Cards: `rounded-2xl` (2rem); chips/pills: `rounded-full`
- Shadows: ambient glow (`shadow-glow-*`), not hard drop shadows
- Typography: Manrope, data values as hero elements, labels uppercase with wide tracking

### uPlot Integration

uPlot is a canvas-based chart library used for all time-series charts. The wrapper handles dark mode, ResizeObserver, multi-series overlays, and uPlot's native zoom/pan.

---

## 11. Authentication

### Token Model

```
Format: 64 hex characters (32 random bytes)
Storage: SQLite auth_tokens table (plaintext — it's an opaque credential, not a password)
Transport: Authorization: Bearer <token> header
Scope: 'read' | 'write' | 'control' | 'admin'
```

Scope hierarchy (each level inherits lower levels):
- **read** — read-only GET access to all data endpoints
- **write** — read + create/update/delete data (journal, measurements, livestock, dosing, maintenance, dashboard, devices, alerts)
- **control** — write + outlet control (`PUT /api/outlets/{id}`) + feed mode (`PUT /api/feed`)
- **admin** — control + token management + config + backup + agent settings

### First-Run Bootstrap

On first start with an empty `auth_tokens` table, the API Server generates a random 32-byte token with `admin` scope, inserts it with label `"default"`, and prints it once to stdout.

### SSE Authentication

`GET /api/stream?token=<token>` — EventSource does not support custom headers in browsers, so the token is passed as a query parameter. Validated with the same SQLite lookup.

---

## 12. Error Handling and Resilience

- **Never crash on external failure.** Apex unreachable, DuckDB write failure, and SQLite errors produce log entries and skip the current operation.
- **Structured errors with context.** All errors are wrapped with `fmt.Errorf("context: %w", err)`.
- **API errors are JSON.** All 4xx and 5xx responses return `{"error":"...", "code":"..."}`.
- **Panics are caught.** The API server middleware catches panics and returns 500.
- **DuckDB writes are transactional.** If any write in the batch fails, the entire poll cycle rolls back.
- **Apex session resilience.** On any 401 from the Apex, the client re-authenticates and retries once.

### Retry Policy

| Operation | Retry? | Strategy |
|---|---|---|
| Apex login | Yes | 3 attempts, 1s backoff |
| Apex status poll | No | Skip cycle, retry on next tick |
| Apex outlet control | No | Return error to caller |
| DuckDB write | No | Skip cycle, log error |
| SQLite read/write | No | Return 500 to API caller |
| ntfy notification | Yes | 2 attempts, 5s backoff |

---

## 13. Data Flows

### Flow 1: Probe Data Collection

```
Apex Hardware → apex.Client.Status() → poller.poll()
  │
  ├── DuckDB: INSERT probe_readings, outlet_states, power_events
  │
  └── internal event bus → SSE broadcaster → browsers
                                           → alert engine
```

### Flow 2: Dashboard Load

```
Browser → GET /api/probes
  ├── DuckDB: SELECT DISTINCT ON (probe_name) latest value per probe
  └── SQLite: SELECT * FROM probe_config (display names, thresholds)
              JOIN at application layer

Browser → GET /api/dashboard
  └── SQLite: SELECT * FROM dashboard_items ORDER BY sort_order
```

### Flow 3: Outlet Control

```
Browser → PUT /api/outlets/1_1 { state: "OFF" }
  ├── Auth middleware: validate token, require control scope
  ├── PUT /rest/outlets/1_1 → Apex Hardware
  ├── SQLite: INSERT INTO events (kind='outlet_changed', initiated_by='ui')
  └── Return updated outlet state
```

### Flow 4: Manual Chemistry Entry

```
Browser → POST /api/measurements { parameter_id, value, measured_at }
  ├── SQLite: INSERT INTO measurements
  └── internal event bus: publishes 'measurement_added' → auto-journal entry
```

---

## 14. NixOS Deployment

### File System Layout

```
/var/lib/symbiont/
├── telemetry.db          # DuckDB — time-series telemetry
├── app.db                # SQLite — app state, config, auth
├── images/               # Livestock and device images
├── frontend/             # Vite build output (static files)
└── backups/
    ├── telemetry-2026-05-01.db
    └── app-2026-05-01.db

/etc/symbiont/
└── env                   # Environment file (secrets, not in nix store)

/run/current-system/sw/bin/
├── symbiont-poller
├── symbiont-api
├── symbiont-mcp
└── symbiont              # CLI
```

### Systemd Services

Four services: `symbiont-poller`, `symbiont-api`, `symbiont-mcp`. Plus systemd timers for nightly backup and weekly retention cleanup. All services run as the `symbiont` system user with `ProtectSystem=strict` and `ReadWritePaths=/var/lib/symbiont`.

---

## 15. Configuration Reference

All configuration comes from environment variables. In development, loaded from `.env` via `godotenv`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `SYMBIONT_APEX_URL` | Yes | — | Apex local IP, e.g. `http://192.168.1.100` |
| `SYMBIONT_APEX_USER` | Yes | — | Apex login username |
| `SYMBIONT_APEX_PASS` | Yes | — | Apex login password |
| `SYMBIONT_DB_PATH` | No | `./telemetry.db` | DuckDB file path |
| `SYMBIONT_SQLITE_PATH` | No | `./app.db` | SQLite file path |
| `SYMBIONT_POLL_INTERVAL` | No | `10s` | Poller interval (Go duration string) |
| `SYMBIONT_API_HOST` | No | `0.0.0.0` | API server bind host |
| `SYMBIONT_API_PORT` | No | `8420` | API server bind port |
| `SYMBIONT_FRONTEND_PATH` | No | `./frontend/dist` | Static frontend files |
| `SYMBIONT_IMAGE_DIR` | No | `./images` | Uploaded images directory |
| `SYMBIONT_BACKUP_PATH` | No | `./backups` | Backup output directory |
| `SYMBIONT_BACKUP_RETAIN` | No | `30` | Number of backup files to keep |
| `SYMBIONT_RETENTION_DAYS` | No | `365` | DuckDB row retention in days |
| `SYMBIONT_API_URL` | No | `http://localhost:8420` | API URL (used by CLI and MCP) |
| `SYMBIONT_TOKEN` | No | — | Auth token (used by CLI and MCP) |
| `SYMBIONT_LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |

---

## 16. Performance Considerations

### DuckDB Write Throughput

At 10-second poll intervals with ~10 probes and ~20 outlets, Symbiont writes approximately 3 rows/second to DuckDB. The single-writer constraint means the Poller holds the connection and writes sequentially.

### DuckDB Read Performance

A 24-hour history query over a single probe (~8,640 rows) typically completes in under 5ms. DuckDB's native `time_bucket` equivalent eliminates application-level aggregation.

### Memory Footprint

Estimated at steady state:
| Process | Estimated RSS |
|---|---|
| symbiont-poller | ~20 MB |
| symbiont-api | ~35 MB |
| symbiont-mcp | ~15 MB |
| Total | ~70 MB |

---

## 17. Security Considerations

### Threat Model

Symbiont runs on a private LAN. Remote access is via Tailscale only — no ports exposed to the internet. The security posture is pragmatic for a home appliance.

### Token Scopes

Tokens are scoped to limit blast radius: a `read` token for monitoring scripts cannot accidentally control outlets. A `write` token for data entry cannot change live hardware. Use the minimum scope needed.

### What's Protected

- All API endpoints require a Bearer token.
- The Apex password lives only in `/etc/symbiont/env` (mode 0400).
- DuckDB and SQLite files are owned by the `symbiont` system user.
- Systemd service hardening: `PrivateTmp`, `NoNewPrivileges`, `ProtectSystem=strict`.

### What's Not Protected

- Tokens stored as plaintext in SQLite. Acceptable given physical security of the host.
- HTTP (not HTTPS) on the local network. Tailscale encrypts traffic end-to-end for remote access.
- No rate limiting. Acceptable for single-user home use.
