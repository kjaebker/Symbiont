# Symbiont — Technical Architecture

> Version 0.1 — Working Document

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
| UI Components | shadcn/ui + Tremor | Unstyled primitives + dashboard-native components |
| Charts | uPlot | Canvas-based, handles millions of points |
| Data fetching | TanStack Query | Cache, background refresh, SSE integration |
| Styling | Tailwind CSS | Utility-first, dark mode by default |

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
│   │  Apex session   │                     │ read                │ │
│   └─────────────────┘                     │                     │ │
│                                           ▼                     │ │
│   ┌─────────────────┐        ┌────────────────────────────────┐ │
│   │  symbiont-api   │◀──────▶│         SQLite                 │ │
│   │                 │  r/w   │    /var/lib/symbiont/           │ │
│   │  :8420          │        │    app.db                      │ │
│   │  REST + SSE     │        └────────────────────────────────┘ │
│   │  Token auth     │                                           │ │
│   └────────┬────────┘                                           │ │
│            │                  ┌────────────────────────────────┐ │
│            │ HTTP/JSON        │       Neptune Apex             │ │
│            │ :8420            │    (local network)             │ │
│            │                  └────────────────────────────────┘ │
│   ┌────────┴────────┐                    ▲                      │ │
│   │  symbiont-mcp   │                    │ outlet control       │ │
│   │                 │                    │ proxied through API  │ │
│   │  MCP protocol   │                    │                      │ │
│   │  Tool surface   │        ┌───────────┴────────────────────┐ │
│   └─────────────────┘        │     symbiont-api               │ │
│                              │  holds Apex session            │ │
│   ┌─────────────────┐        │  proxies PUT /rest/outlets     │ │
│   │  symbiont (CLI) │        └────────────────────────────────┘ │
│   │                 │                                           │ │
│   │  Cobra commands │                                           │ │
│   │  JSON output    │                                           │ │
│   └─────────────────┘                                           │ │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    Frontend (static)                     │   │
│   │   Served by symbiont-api or nginx on same host           │   │
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
module github.com/yourname/symbiont

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
-- Polled infrequently (or on change detection)
CREATE TABLE controller_meta (
    ts          TIMESTAMPTZ NOT NULL,
    serial      VARCHAR,
    firmware    VARCHAR,
    hardware    VARCHAR,
    PRIMARY KEY (ts)
);
```

**Key query patterns against DuckDB:**

```sql
-- Current value of all probes (latest row per probe_name)
SELECT DISTINCT ON (probe_name)
    probe_name, probe_type, value, unit, ts
FROM probe_readings
ORDER BY probe_name, ts DESC;

-- Time-bucketed history for a single probe
-- 1-minute buckets over the last 24 hours
SELECT
    time_bucket(INTERVAL '1 minute', ts) AS bucket,
    AVG(value) AS value
FROM probe_readings
WHERE probe_name = $1
  AND ts >= NOW() - INTERVAL '24 hours'
GROUP BY bucket
ORDER BY bucket;

-- Outlet state transitions (change detection)
SELECT
    ts,
    outlet_id,
    outlet_name,
    state,
    LAG(state) OVER (PARTITION BY outlet_id ORDER BY ts) AS prev_state
FROM outlet_states
WHERE outlet_id = $1
  AND ts >= $2
HAVING state != prev_state OR prev_state IS NULL
ORDER BY ts;
```

**DuckDB connection management in Go:**

DuckDB enforces a single writer. The Poller opens a read-write connection and holds it for the lifetime of the process. The API Server opens a read-only connection. This is enforced at the connection string level:

```go
// Poller — read-write, sole writer
db, err := sql.Open("duckdb", "/var/lib/symbiont/telemetry.db")

// API Server — read-only, multiple readers allowed
db, err := sql.Open("duckdb", "/var/lib/symbiont/telemetry.db?access_mode=read_only")
```

### SQLite Schema

```sql
PRAGMA journal_mode=WAL;   -- Enable WAL for concurrent reads
PRAGMA foreign_keys=ON;

-- Auth tokens
CREATE TABLE auth_tokens (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    token       TEXT     NOT NULL UNIQUE,
    label       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used   DATETIME
);

-- Per-probe display and threshold configuration
CREATE TABLE probe_config (
    probe_name      TEXT PRIMARY KEY,
    display_name    TEXT,
    unit_override   TEXT,
    display_order   INTEGER NOT NULL DEFAULT 999,
    min_normal      REAL,
    max_normal      REAL,
    min_warning     REAL,
    max_warning     REAL,
    hidden          INTEGER NOT NULL DEFAULT 0
);

-- Per-outlet display configuration
CREATE TABLE outlet_config (
    outlet_id       TEXT PRIMARY KEY,
    display_name    TEXT,
    display_order   INTEGER NOT NULL DEFAULT 999,
    icon            TEXT,
    hidden          INTEGER NOT NULL DEFAULT 0
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

-- Notification delivery targets
CREATE TABLE notification_targets (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    type        TEXT     NOT NULL,
    config      TEXT     NOT NULL,  -- JSON: {url, topic, priority, ...}
    label       TEXT,
    enabled     INTEGER  NOT NULL DEFAULT 1
);

-- Alert firing history (for deduplication and display)
CREATE TABLE alert_events (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    rule_id         INTEGER  NOT NULL REFERENCES alert_rules(id),
    fired_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cleared_at      DATETIME,
    peak_value      REAL,
    notified        INTEGER  NOT NULL DEFAULT 0
);

-- Outlet change log
CREATE TABLE outlet_event_log (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    ts              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    outlet_id       TEXT     NOT NULL,
    outlet_name     TEXT,
    from_state      TEXT,
    to_state        TEXT     NOT NULL,
    initiated_by    TEXT     NOT NULL CHECK(initiated_by IN ('ui','cli','mcp','api','apex'))
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

-- Indexes
CREATE INDEX idx_alert_events_rule ON alert_events(rule_id, fired_at DESC);
CREATE INDEX idx_outlet_event_log_ts ON outlet_event_log(ts DESC);
CREATE INDEX idx_outlet_event_log_outlet ON outlet_event_log(outlet_id, ts DESC);
```

---

## 4. Apex Client

The Apex client (`internal/apex/client.go`) is the only component that communicates with the Neptune Apex hardware. It is used by both the Poller (for status reads) and the API Server (for outlet control). Each process holds its own client instance with its own session.

### Session Lifecycle

```
                ┌─────────────┐
                │    Start    │
                └──────┬──────┘
                       │
                       ▼
              ┌────────────────┐
              │  POST /rest/   │
              │    /login      │
              └───────┬────────┘
                      │ 200 + Set-Cookie: connect.sid
                      ▼
              ┌────────────────┐
       ┌─────▶│  Session OK    │◀──────────────┐
       │      └───────┬────────┘               │
       │              │                        │
       │    Poll / Command request              │
       │              │                        │
       │              ▼                        │
       │      ┌────────────────┐               │
       │      │  401 Response? │──── No ───────┘
       │      └───────┬────────┘
       │              │ Yes
       │              ▼
       │      ┌────────────────┐
       └──────│  Re-authenticate│
              └────────────────┘
```

### Client Interface

```go
// internal/apex/client.go

type Client interface {
    Status(ctx context.Context) (*StatusResponse, error)
    Outlets(ctx context.Context) ([]Outlet, error)
    SetOutlet(ctx context.Context, id string, state OutletState) (*Outlet, error)
}

type client struct {
    baseURL    string
    username   string
    password   string
    httpClient *http.Client
    sessionMu  sync.Mutex
    cookie     *http.Cookie
}

// NewClient creates and immediately authenticates a client.
func NewClient(baseURL, username, password string) (Client, error)

// login performs the auth POST and stores the session cookie.
func (c *client) login(ctx context.Context) error

// do executes a request, automatically re-authenticating on 401.
func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error)
```

### Data Models

```go
// internal/apex/models.go

type StatusResponse struct {
    System   SystemInfo  `json:"system"`
    Inputs   []Input     `json:"inputs"`
    Outputs  []Output    `json:"outputs"`
}

type SystemInfo struct {
    Serial        string `json:"serial"`
    Hostname      string `json:"hostname"`
    Firmware      string `json:"firmware"`
    Hardware      string `json:"hardware"`
    PowerFailed   string `json:"power_failed"`
    PowerRestored string `json:"power_restored"`
    Date          string `json:"date"`
}

type Input struct {
    Name  string  `json:"name"`
    Value float64 `json:"value"`
    Unit  string  `json:"unit"`
    Type  string  `json:"type"`
}

type Output struct {
    DID    string  `json:"did"`
    Name   string  `json:"name"`
    State  string  `json:"state"`
    Xstatus string `json:"xstatus"`
    Watts  float64 `json:"watts"`
    Amps   float64 `json:"amps"`
}

type OutletState string
const (
    OutletOn  OutletState = "ON"
    OutletOff OutletState = "OFF"
    // AUTO is not supported by the Apex REST API — must use Apex web UI.
)
```

> **Note:** The exact field names and JSON structure must be verified via DevTools capture against your specific AOS firmware version. The above is informed by community reverse-engineering and may require adjustment. This is Phase 1's first task.

---

## 5. Poller Service

The Poller is a long-running Go binary whose sole job is polling the Apex every 10 seconds and appending rows to DuckDB. It has no HTTP server, no config API, and no external interface.

### Architecture

```go
// cmd/poller/main.go

func main() {
    cfg := config.Load()
    apex := apex.NewClient(cfg.ApexURL, cfg.ApexUser, cfg.ApexPass)
    db := duckdb.Open(cfg.DBPath)
    poller := poller.New(apex, db, cfg.PollInterval)
    
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer cancel()
    
    poller.Run(ctx)  // blocks until ctx is done
}
```

### Polling Loop

```go
// internal/poller/poller.go

type Poller struct {
    apex     apex.Client
    db       *duckdb.DB
    interval time.Duration
    logger   *slog.Logger
}

func (p *Poller) Run(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    // Poll immediately on start, then on each tick
    p.poll(ctx)

    for {
        select {
        case <-ticker.C:
            p.poll(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (p *Poller) poll(ctx context.Context) {
    status, err := p.apex.Status(ctx)
    if err != nil {
        p.logger.Error("apex poll failed", "err", err)
        // Do not write partial data. Skip this cycle.
        return
    }

    ts := time.Now().UTC()

    if err := p.db.WriteProbeReadings(ctx, ts, status.Inputs); err != nil {
        p.logger.Error("duckdb write failed", "table", "probe_readings", "err", err)
    }

    if err := p.db.WriteOutletStates(ctx, ts, status.Outputs); err != nil {
        p.logger.Error("duckdb write failed", "table", "outlet_states", "err", err)
    }

    p.db.WritePowerEvents(ctx, ts, status.System)      // no-ops if no new events
    p.db.WriteControllerMeta(ctx, ts, status.System)   // no-ops if unchanged
}
```

### Failure Behavior

- On Apex unreachable: log error, skip cycle, retry next tick. No crash.
- On DuckDB write failure: log error, continue. Row is skipped — no retry, no buffer. A 10-second gap in telemetry is acceptable.
- On 401 from Apex: client re-authenticates automatically (see Apex Client). Transparent to the Poller.
- On SIGTERM: graceful shutdown via context cancellation. In-flight poll completes before exit.

### Structured Logging

All services use Go's `log/slog` with JSON output. This makes log aggregation and structured querying easy from the NixOS journal.

```go
slog.Info("poll complete",
    "duration_ms", elapsed.Milliseconds(),
    "probes", len(status.Inputs),
    "outlets", len(status.Outputs),
)
```

---

## 6. API Server

The API Server is a Go HTTP server serving the REST API, SSE stream, and static frontend files. It reads from DuckDB (read-only), reads and writes SQLite, and proxies outlet control commands to the Apex.

### Routing

```
GET  /api/probes                          → probes.HandleList
GET  /api/probes/:name/history            → probes.HandleHistory
GET  /api/outlets                         → outlets.HandleList
PUT  /api/outlets/:id                     → outlets.HandleSet
GET  /api/outlets/:id/events              → outlets.HandleEvents
GET  /api/system                          → system.HandleStatus
GET  /api/stream                          → sse.HandleStream
GET  /api/alerts                          → alerts.HandleList
POST /api/alerts                          → alerts.HandleCreate
PUT  /api/alerts/:id                      → alerts.HandleUpdate
DEL  /api/alerts/:id                      → alerts.HandleDelete
GET  /api/config/probes                   → config.HandleProbeList
PUT  /api/config/probes/:name             → config.HandleProbeUpdate
GET  /api/config/outlets                  → config.HandleOutletList
PUT  /api/config/outlets/:id              → config.HandleOutletUpdate
POST /api/auth/tokens                     → auth.HandleCreate
GET  /api/auth/tokens                     → auth.HandleList
DEL  /api/auth/tokens/:id                 → auth.HandleDelete
POST /api/system/backup                   → system.HandleBackup

GET  /*                                   → static file server (frontend build)
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
Auth (validate Bearer token, skip /api/stream for SSE clients)
  │
  ▼
Handler
```

### Request / Response Shapes

All responses are JSON. Error responses follow a consistent shape:

```json
{
  "error": "human-readable message",
  "code":  "machine_readable_code"
}
```

**GET /api/probes**

```json
{
  "probes": [
    {
      "name":         "Temp",
      "display_name": "Temperature",
      "type":         "temp",
      "value":        78.4,
      "unit":         "F",
      "ts":           "2025-03-20T14:32:00Z",
      "status":       "normal"
    }
  ],
  "polled_at": "2025-03-20T14:32:01Z"
}
```

`status` is derived at the API layer by comparing `value` against `probe_config` thresholds from SQLite. Values: `"normal"`, `"warning"`, `"critical"`, `"unknown"` (no thresholds configured).

**GET /api/probes/:name/history**

Query params:
- `from` — ISO 8601 timestamp (default: 24 hours ago)
- `to` — ISO 8601 timestamp (default: now)
- `interval` — bucket size: `10s`, `1m`, `5m`, `15m`, `1h`, `1d` (default: auto based on range)

```json
{
  "probe":    "Temp",
  "from":     "2025-03-19T14:32:00Z",
  "to":       "2025-03-20T14:32:00Z",
  "interval": "5m",
  "data": [
    { "ts": "2025-03-19T14:35:00Z", "value": 78.2 },
    { "ts": "2025-03-19T14:40:00Z", "value": 78.4 }
  ]
}
```

Auto-interval selection based on range:
| Range | Default Interval |
|---|---|
| ≤ 2 hours | 10s (raw) |
| ≤ 12 hours | 1m |
| ≤ 3 days | 5m |
| ≤ 2 weeks | 15m |
| ≤ 2 months | 1h |
| > 2 months | 1d |

**GET /api/outlets**

```json
{
  "outlets": [
    {
      "id":           "1_1",
      "name":         "Return Pump",
      "display_name": "Return Pump",
      "state":        "AON",
      "watts":        48.2,
      "amps":         0.41
    }
  ]
}
```

**PUT /api/outlets/:id**

Request body:
```json
{ "state": "OFF" }
```

Response:
```json
{
  "id":       "1_1",
  "name":     "Return Pump",
  "state":    "OFF",
  "xstatus":  "OFF",
  "watts":    0,
  "amps":     0,
  "logged_at": "2025-03-20T14:33:00Z"
}
```

This handler: validates the state value, sends `PUT /rest/outlets/:id` to the Apex, writes a row to `outlet_event_log` in SQLite, and returns the updated outlet.

**GET /api/system**

```json
{
  "controller": {
    "serial":   "AC5:12345",
    "firmware": "5.08A_7A18",
    "hardware": "1.0"
  },
  "poller": {
    "last_poll_ts": "2025-03-20T14:32:01Z",
    "poll_ok":      true,
    "poll_interval_seconds": 10
  },
  "db": {
    "duckdb_size_bytes": 134217728,
    "sqlite_size_bytes": 1048576
  }
}
```

**GET /api/stream (Server-Sent Events)**

Stream format:
```
event: probe_update
data: {"probes":[...]}

event: outlet_update
data: {"outlets":[...]}

event: alert_fired
data: {"rule_id":3,"probe":"pH","value":7.8,"severity":"warning"}

event: heartbeat
data: {"ts":"2025-03-20T14:32:01Z"}
```

The SSE handler pushes on each poller cycle (every 10s) plus a heartbeat every 30s to detect dropped connections. Clients that disconnect are removed from the broadcaster set immediately.

### SSE Broadcaster

```go
// internal/api/sse.go

type Broadcaster struct {
    mu      sync.RWMutex
    clients map[string]chan Event  // client_id → channel
}

func (b *Broadcaster) Subscribe(id string) <-chan Event
func (b *Broadcaster) Unsubscribe(id string)
func (b *Broadcaster) Publish(e Event)
```

The Poller and Alert Engine publish events to the Broadcaster. The SSE handler subscribes per connected client. The Broadcaster never blocks — slow clients are dropped.

---

## 7. CLI

The CLI binary (`symbiont`) is a Cobra-based command-line tool. It is a pure HTTP client of the API server. It never accesses the databases directly.

### Command Tree

```
symbiont
├── probes
│   ├── current               List current probe values
│   └── history <name>        Time-series history for one probe
│       --from  ISO timestamp
│       --to    ISO timestamp
│       --interval [10s|1m|5m|15m|1h|1d]
│
├── outlets
│   ├── list                  List all outlet states
│   └── set <id> <ON|OFF>       Control an outlet (AUTO via Apex web UI only)
│
├── alerts
│   ├── list                  List alert rules
│   ├── create                Create an alert rule (interactive or flags)
│   ├── update <id>           Update a rule
│   ├── delete <id>           Delete a rule
│   └── events                Recent alert firings
│
├── system
│   ├── status                Controller + poller health
│   └── backup                Trigger a manual backup
│
└── auth
    ├── tokens list
    ├── tokens create --label <label>
    └── tokens revoke <id>
```

Global flags available on every command:
```
--json          Output raw JSON (for scripting and agent use)
--api-url       Override API base URL (default: http://localhost:8420)
--token         Override auth token (default: from SYMBIONT_TOKEN env or ~/.config/symbiont/token)
```

### Output Modes

Human-readable (default):
```
$ symbiont probes current

PROBE         VALUE    UNIT    STATUS    UPDATED
Temp          78.4     F       normal    14:32:01
pH            8.21     pH      normal    14:32:01
ORP           380      mV      warning   14:32:01
Salinity      35.2     ppt     normal    14:32:01
```

JSON (`--json`):
```json
{
  "probes": [
    { "name": "Temp", "value": 78.4, "unit": "F", "status": "normal", "ts": "..." }
  ]
}
```

---

## 8. MCP Server

The MCP Server exposes Symbiont's data and control capabilities to AI assistants via the Model Context Protocol. It is a separate binary running as a systemd service, communicating with the API Server over localhost.

### Tool Surface

```go
// internal/mcp/tools.go

// Tools registered with the MCP server:

"get_current_parameters"
  Description: "Get the current value of all probes (temperature, pH, ORP, salinity, etc.)"
  Input:  none
  Output: JSON matching GET /api/probes response

"get_probe_history"
  Description: "Get time-series history for a specific probe"
  Input:  { name: string, from?: ISO, to?: ISO, interval?: string }
  Output: JSON matching GET /api/probes/:name/history response

"get_outlet_states"
  Description: "Get the current state of all outlets and their power draw"
  Input:  none
  Output: JSON matching GET /api/outlets response

"control_outlet"
  Description: "Set an outlet to ON or OFF (AUTO not supported by Apex REST API)"
  Input:  { id: string, state: "ON" | "OFF" }
  Output: JSON matching PUT /api/outlets/:id response

"get_outlet_event_log"
  Description: "Get recent outlet changes — who changed what and when"
  Input:  { outlet_id?: string, limit?: int }
  Output: JSON matching GET /api/outlets/:id/events response

"get_alert_rules"
  Description: "Get configured alert thresholds for all probes"
  Input:  none
  Output: JSON matching GET /api/alerts response

"get_system_status"
  Description: "Get controller metadata and poller health"
  Input:  none
  Output: JSON matching GET /api/system response

"summarize_tank_health"
  Description: "Get a comprehensive summary of current tank state across all parameters"
  Input:  none
  Output: Calls get_current_parameters + get_outlet_states + get_system_status,
          synthesizes into a single structured health object
```

### MCP Server Implementation

```go
// cmd/mcp/main.go

func main() {
    cfg := config.Load()
    apiClient := apiclient.New(cfg.APIURL, cfg.Token)

    s := mcp.NewServer("symbiont", "1.0.0")
    tools.Register(s, apiClient)

    if err := s.ServeStdio(); err != nil {
        log.Fatal(err)
    }
}
```

The MCP server communicates via stdio (stdin/stdout) per the MCP protocol — it does not bind to a network port. Claude Desktop and Claude Code connect to it by launching the binary as a subprocess.

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

---

## 9. Alert Engine

The Alert Engine runs as a background goroutine inside the API Server process. It evaluates alert rules against the latest probe readings on every SSE broadcast cycle (i.e., every 10 seconds when the Poller publishes an update).

### Alert Evaluation

```go
// internal/alerts/engine.go

type Engine struct {
    sqlite   *sqlite.DB
    notifier notify.Notifier
    state    map[int]AlertState  // rule_id → current state
    mu       sync.Mutex
}

type AlertState struct {
    Active    bool
    FiredAt   time.Time
    PeakValue float64
}

func (e *Engine) Evaluate(probes []api.Probe) {
    rules := e.sqlite.GetEnabledAlertRules()

    for _, rule := range rules {
        probe := findProbe(probes, rule.ProbeName)
        if probe == nil {
            continue
        }

        breached := e.isBreached(rule, probe.Value)
        state := e.state[rule.ID]

        if breached && !state.Active {
            // Alert just fired
            e.fire(rule, probe)
        } else if !breached && state.Active {
            // Alert cleared
            e.clear(rule)
        } else if breached && state.Active {
            // Ongoing — update peak value
            e.updatePeak(rule, probe.Value)
        }
    }
}
```

### Debounce / Cooldown

Each rule has a `cooldown_minutes` field. After firing, the engine does not re-fire the same rule until the cooldown expires, even if the value dips below threshold and rises again. This prevents notification spam during oscillating values.

### Notification Interface

```go
// internal/notify/notifier.go

type Notifier interface {
    Send(ctx context.Context, n Notification) error
}

type Notification struct {
    Title    string
    Body     string
    Priority string  // "default" | "high" | "urgent"
    Tags     []string
}
```

The `ntfy.go` implementation POSTs to an ntfy.sh topic URL. Additional implementations (webhook, etc.) can be dropped in without changing the engine.

---

## 10. Frontend Architecture

### Page Structure

```
App
├── Layout
│   ├── Sidebar (nav)
│   └── TopBar (system status badge, last poll indicator)
│
├── /dashboard           → Dashboard
│   ├── ProbeGrid        → ProbeCard[] (Tremor stat + sparkline)
│   └── OutletGrid       → OutletCard[] (state badge + toggle)
│
├── /history             → History
│   ├── ProbeSelector    → multi-select dropdown
│   ├── TimeRangePicker  → preset + custom range
│   └── ProbeChart       → uPlot wrapper (multi-series)
│
├── /outlets             → Outlets
│   ├── OutletTable      → full outlet list with controls
│   └── EventLog         → recent outlet changes
│
├── /alerts              → Alerts
│   ├── AlertRuleList    → existing rules
│   └── AlertRuleForm    → create / edit rule (shadcn dialog)
│
└── /settings            → Settings
    ├── ProbeConfigTable → display names, unit overrides, ordering
    ├── OutletConfigTable→ display names, ordering
    ├── TokenManager     → list, create, revoke tokens
    ├── NotificationConfig → ntfy.sh setup
    └── BackupStatus     → last backup, manual trigger
```

### State Management

No global state store (no Zustand, no Redux). State lives at two levels:

**Server state** — managed by TanStack Query:
```typescript
// hooks/useProbes.ts
export function useProbes() {
  return useQuery({
    queryKey: ['probes'],
    queryFn: () => api.getProbes(),
    staleTime: 10_000,   // match poll interval
    refetchInterval: false, // SSE drives updates, not polling
  });
}
```

**SSE invalidation** — the SSE hook invalidates TanStack Query caches on each server event:
```typescript
// hooks/useSSE.ts
export function useSSE() {
  const queryClient = useQueryClient();

  useEffect(() => {
    const es = new EventSource('/api/stream', {
      headers: { Authorization: `Bearer ${token}` }
    });

    es.addEventListener('probe_update', () => {
      queryClient.invalidateQueries({ queryKey: ['probes'] });
    });

    es.addEventListener('outlet_update', () => {
      queryClient.invalidateQueries({ queryKey: ['outlets'] });
    });

    return () => es.close();
  }, []);
}
```

**Local UI state** — React `useState` for form inputs, modal open/close, selected time ranges.

### uPlot Integration

uPlot is a canvas-based chart library that handles millions of data points without frame drops. The wrapper component handles:

- Dark mode color scheme (CSS variable driven)
- Consistent axis formatting (time, value with units)
- Window resize with ResizeObserver
- Multi-series overlay (multiple probes on one chart)
- Zoom and pan (uPlot native)

```typescript
// components/ProbeChart.tsx

interface ProbeChartProps {
  series: Array<{
    name: string;
    data: Array<{ ts: string; value: number }>;
    unit: string;
    color: string;
  }>;
  height?: number;
}

export function ProbeChart({ series, height = 300 }: ProbeChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<uPlot | null>(null);

  // Build uPlot data format: [timestamps[], series1values[], series2values[], ...]
  const data = useMemo(() => buildUPlotData(series), [series]);
  const opts = useMemo(() => buildUPlotOpts(series, height), [series, height]);

  useEffect(() => {
    if (!containerRef.current) return;
    chartRef.current = new uPlot(opts, data, containerRef.current);
    return () => chartRef.current?.destroy();
  }, []);

  // Update data without re-creating the chart
  useEffect(() => {
    chartRef.current?.setData(data);
  }, [data]);

  return <div ref={containerRef} />;
}
```

### Component Hierarchy

| Layer | Library | Usage |
|---|---|---|
| Primitives | shadcn/ui | Button, Input, Dialog, Select, Table, Badge, Switch |
| Dashboard | Tremor | Card, Metric, Sparkline, BadgeDelta |
| Charts | uPlot | All time-series (wrapped in ProbeChart) |
| Icons | lucide-react | Consistent icon set |
| Layout | Tailwind CSS | All spacing, color, responsive behavior |

### Dark Mode

Dark mode is the default. Implemented via Tailwind's `class` strategy:

```html
<html class="dark">
```

The `dark:` prefix is used throughout. A light mode toggle is deferred — the implementation is not architecturally blocked, it's just not a priority.

---

## 11. Authentication

### Token Model

```
Token format: 64 hex characters (32 random bytes)
Example: a3f8e2c1d7b4...

Storage: SQLite auth_tokens table (token stored as plaintext — it's an opaque credential, not a password)
Transport: Authorization: Bearer <token> header on all API requests
```

### First-Run Bootstrap

On first start, if `auth_tokens` table is empty, the API Server:
1. Generates a random 32-byte token
2. Inserts it into SQLite with label `"default"`
3. Prints it to stdout once:
   ```
   Symbiont API token (save this): a3f8e2c1d7b4...
   ```

This token is never shown again. If lost, revoke all tokens via the CLI with direct DB access (special `--db-path` flag on `symbiont auth reset`) and restart.

### Token Validation Middleware

```go
func AuthMiddleware(sqlite *sqlite.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearer(r.Header.Get("Authorization"))
            if token == "" {
                writeError(w, 401, "missing token")
                return
            }

            ok, id := sqlite.ValidateToken(token)
            if !ok {
                writeError(w, 401, "invalid token")
                return
            }

            // Update last_used async — don't block the request
            go sqlite.TouchToken(id)

            next.ServeHTTP(w, r)
        })
    }
}
```

### SSE Authentication

EventSource does not support custom headers in browsers. The SSE endpoint uses a token passed as a query parameter instead:

```
GET /api/stream?token=<token>
```

The SSE handler validates the query parameter token using the same SQLite lookup. This endpoint must not be proxied publicly without Tailscale or similar.

---

## 12. Error Handling and Resilience

### Philosophy

- **Never crash on external failure.** Apex unreachable, DuckDB write failure, and SQLite errors all produce log entries and skip the current operation. They do not take down the service.
- **Structured errors with context.** All errors are wrapped with `fmt.Errorf("context: %w", err)` to preserve stack context in logs.
- **API errors are JSON.** All 4xx and 5xx responses return `{"error":"...", "code":"..."}`.
- **Panics are caught.** The API server middleware catches panics and returns 500. The Poller uses `defer recover()` in the poll goroutine.

### Apex Session Resilience

The Apex session can expire without warning. The client handles this transparently:

```go
func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
    resp, err := c.send(ctx, req)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode == http.StatusUnauthorized {
        resp.Body.Close()
        if err := c.login(ctx); err != nil {
            return nil, fmt.Errorf("re-auth failed: %w", err)
        }
        // Retry original request once with new session
        return c.send(ctx, req)
    }

    return resp, nil
}
```

### DuckDB Write Resilience

DuckDB writes happen in a transaction per poll cycle. If any write in the batch fails, the transaction rolls back and the entire poll cycle is skipped. Partial writes do not corrupt the dataset.

```go
func (db *DB) WritePollCycle(ctx context.Context, ts time.Time, status *apex.StatusResponse) error {
    tx, err := db.conn.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()  // no-op if committed

    if err := writeProbeReadings(tx, ts, status.Inputs); err != nil {
        return fmt.Errorf("write probes: %w", err)
    }

    if err := writeOutletStates(tx, ts, status.Outputs); err != nil {
        return fmt.Errorf("write outlets: %w", err)
    }

    return tx.Commit()
}
```

### Retry Policy

| Operation | Retry? | Strategy |
|---|---|---|
| Apex login | Yes | 3 attempts, 1s backoff |
| Apex status poll | No | Skip cycle, retry on next tick |
| Apex outlet control | No | Return error to caller |
| DuckDB write | No | Skip cycle, log error |
| SQLite read | No | Return 500 to API caller |
| SQLite write | No | Return 500 to API caller |
| ntfy notification | Yes | 2 attempts, 5s backoff |

---

## 13. Data Flows

### Flow 1: Probe Data Collection

```
Apex Hardware
  │
  │  GET /rest/status (HTTP, every 10s)
  ▼
apex.Client.Status()
  │
  │  *apex.StatusResponse
  ▼
poller.poll()
  │
  │  BeginTx
  ├──▶ INSERT INTO probe_readings
  ├──▶ INSERT INTO outlet_states
  ├──▶ INSERT INTO power_events (conditional)
  │    Commit
  ▼
DuckDB telemetry.db
  │
  │  Broadcaster.Publish(Event{Type: "probe_update"})
  ▼
SSE Broadcaster
  │
  │  chan Event (per connected client)
  ▼
Browser (SSE)
  │
  │  queryClient.invalidateQueries(['probes'])
  ▼
React re-render (TanStack Query refetch → GET /api/probes)
```

### Flow 2: Dashboard Load

```
Browser → GET /api/probes
            │
            ├── DuckDB: SELECT DISTINCT ON (probe_name) ... ORDER BY ts DESC
            │                     (latest value per probe)
            │
            └── SQLite: SELECT * FROM probe_config
                                  (display names, thresholds)
                                  JOIN at application layer

         → GET /api/outlets
            │
            ├── DuckDB: SELECT DISTINCT ON (outlet_id) ... ORDER BY ts DESC
            │
            └── SQLite: SELECT * FROM outlet_config
```

### Flow 3: Outlet Control

```
Browser → PUT /api/outlets/1_1 { state: "OFF" }
            │
            ├── Validate token (SQLite)
            ├── Validate state value
            │
            ├── PUT /rest/outlets/1_1 → Apex Hardware
            │     { state: "OFF" }
            │
            ├── SQLite: INSERT INTO outlet_event_log
            │     (outlet_id, from_state, to_state, initiated_by='ui')
            │
            └── Return updated outlet state to browser
```

### Flow 4: History Query

```
Browser → GET /api/probes/Temp/history?from=...&to=...&interval=5m
            │
            └── DuckDB:
                  SELECT
                    time_bucket(INTERVAL '5 minutes', ts) AS bucket,
                    AVG(value) AS value
                  FROM probe_readings
                  WHERE probe_name = 'Temp'
                    AND ts BETWEEN $from AND $to
                  GROUP BY bucket
                  ORDER BY bucket
```

### Flow 5: Alert Evaluation

```
Poller publishes SSE event
  │
  ▼
Alert Engine (goroutine in API server)
  │
  ├── SQLite: SELECT * FROM alert_rules WHERE enabled = 1
  │
  ├── Compare latest probe values against each rule
  │
  ├── If newly breached:
  │   ├── SQLite: INSERT INTO alert_events
  │   └── notify.Send() → POST to ntfy.sh
  │
  └── If cleared:
      └── SQLite: UPDATE alert_events SET cleared_at = NOW()
```

---

## 14. NixOS Deployment

### File System Layout

```
/var/lib/symbiont/
├── telemetry.db          # DuckDB — time-series telemetry
├── app.db                # SQLite — app state, config, auth
├── frontend/             # Vite build output (static files)
└── backups/
    ├── telemetry-2025-03-19.db
    └── app-2025-03-19.db

/etc/symbiont/
└── env                   # Environment file (secrets, not in nix store)

/run/current-system/sw/bin/
├── symbiont-poller
├── symbiont-api
├── symbiont-mcp
└── symbiont              # CLI
```

### Environment File

`/etc/symbiont/env` (mode 0400, owned by symbiont system user):

```bash
SYMBIONT_APEX_URL=http://192.168.1.100
SYMBIONT_APEX_USER=admin
SYMBIONT_APEX_PASS=your-apex-password
SYMBIONT_DB_PATH=/var/lib/symbiont/telemetry.db
SYMBIONT_SQLITE_PATH=/var/lib/symbiont/app.db
SYMBIONT_POLL_INTERVAL=10s
SYMBIONT_API_PORT=8420
SYMBIONT_BACKUP_PATH=/var/lib/symbiont/backups
SYMBIONT_BACKUP_RETAIN=30
SYMBIONT_API_URL=http://localhost:8420  # used by MCP and CLI
SYMBIONT_TOKEN=                          # set by first-run, or manually
```

### Systemd Services (flake.nix)

```nix
systemd.services.symbiont-poller = {
  description = "Symbiont Poller — Neptune Apex data collection";
  wantedBy = [ "multi-user.target" ];
  after = [ "network.target" ];
  serviceConfig = {
    Type = "simple";
    ExecStart = "${pkgs.symbiont-poller}/bin/poller";
    Restart = "always";
    RestartSec = "5s";
    EnvironmentFile = "/etc/symbiont/env";
    User = "symbiont";
    Group = "symbiont";
    StateDirectory = "symbiont";
    # Hardening
    PrivateTmp = true;
    NoNewPrivileges = true;
    ProtectSystem = "strict";
    ReadWritePaths = [ "/var/lib/symbiont" ];
  };
};

systemd.services.symbiont-api = {
  description = "Symbiont API Server";
  wantedBy = [ "multi-user.target" ];
  after = [ "network.target" "symbiont-poller.service" ];
  serviceConfig = {
    Type = "simple";
    ExecStart = "${pkgs.symbiont-api}/bin/api";
    Restart = "always";
    RestartSec = "3s";
    EnvironmentFile = "/etc/symbiont/env";
    User = "symbiont";
    Group = "symbiont";
    StateDirectory = "symbiont";
    PrivateTmp = true;
    NoNewPrivileges = true;
    ProtectSystem = "strict";
    ReadWritePaths = [ "/var/lib/symbiont" ];
  };
};

systemd.services.symbiont-mcp = {
  description = "Symbiont MCP Server";
  wantedBy = [ "multi-user.target" ];
  after = [ "symbiont-api.service" ];
  serviceConfig = {
    Type = "simple";
    ExecStart = "${pkgs.symbiont-mcp}/bin/mcp";
    Restart = "always";
    EnvironmentFile = "/etc/symbiont/env";
    User = "symbiont";
    Group = "symbiont";
  };
};

# Nightly backup
systemd.services.symbiont-backup = {
  description = "Symbiont nightly backup";
  serviceConfig = {
    Type = "oneshot";
    ExecStart = "${pkgs.symbiont}/bin/symbiont system backup";
    EnvironmentFile = "/etc/symbiont/env";
    User = "symbiont";
  };
};
systemd.timers.symbiont-backup = {
  wantedBy = [ "timers.target" ];
  timerConfig = {
    OnCalendar = "daily";
    Persistent = true;
  };
};

# Weekly retention cleanup
systemd.services.symbiont-cleanup = {
  description = "Symbiont telemetry retention cleanup";
  serviceConfig = {
    Type = "oneshot";
    ExecStart = "${pkgs.symbiont}/bin/symbiont system cleanup";
    EnvironmentFile = "/etc/symbiont/env";
    User = "symbiont";
  };
};
systemd.timers.symbiont-cleanup = {
  wantedBy = [ "timers.target" ];
  timerConfig = {
    OnCalendar = "weekly";
    Persistent = true;
  };
};
```

---

## 15. Configuration Reference

All configuration is read from environment variables (loaded from `/etc/symbiont/env` via systemd `EnvironmentFile` in production, or from `.env` file in development via `godotenv`).

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
| `SYMBIONT_BACKUP_PATH` | No | `./backups` | Backup output directory |
| `SYMBIONT_BACKUP_RETAIN` | No | `30` | Number of backup files to keep |
| `SYMBIONT_RETENTION_DAYS` | No | `365` | DuckDB row retention in days |
| `SYMBIONT_API_URL` | No | `http://localhost:8420` | API URL (used by CLI and MCP) |
| `SYMBIONT_TOKEN` | No | — | Auth token (used by CLI and MCP) |
| `SYMBIONT_LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |

---

## 16. Performance Considerations

### DuckDB Write Throughput

At 10-second poll intervals with ~10 probes and ~20 outlets, Symbiont writes approximately 3 rows/second to DuckDB. This is negligible for DuckDB. Even at 1-second intervals it would not stress the engine.

The single-writer constraint means the Poller holds the connection and writes sequentially. No connection pooling, no concurrency needed.

### DuckDB Read Performance

The API Server opens a read-only DuckDB connection and reuses it for the process lifetime. DuckDB's columnar format makes time-range scans extremely fast — a 24-hour history query over a single probe (~8,640 rows) typically completes in under 5ms.

For the history endpoint with bucketing, DuckDB's native `time_bucket` equivalent (using `epoch` math or date_trunc) eliminates the need for any application-level aggregation.

### uPlot Data Volume

The frontend can comfortably render 10,000+ data points per series in uPlot without frame drops. At 1-minute intervals, a 7-day history is 10,080 points — handled trivially. At 10-second raw intervals, a 24-hour history is 8,640 points — also fine.

Auto-interval selection on the history endpoint prevents the API from returning more points than the frontend can usefully display.

### SSE Connection Overhead

Each connected browser tab holds one SSE connection (one goroutine + channel in the Broadcaster). At home usage this is 1-3 connections. The overhead is negligible.

### Memory Footprint

Estimated at steady state:
| Process | Estimated RSS |
|---|---|
| symbiont-poller | ~20 MB |
| symbiont-api | ~30 MB |
| symbiont-mcp | ~15 MB |
| Total | ~65 MB |

This is well within the headroom on the NixOS mini PC.

---

## 17. Security Considerations

### Threat Model

Symbiont runs on a private LAN behind a home router. The threat model is:

1. **Other LAN devices** should not be able to toggle reef equipment without the token.
2. **Remote access** is via Tailscale only — no ports exposed to the internet.
3. **Physical access** to the mini PC is out of scope.

This is not a hardened multi-tenant system. The security posture is pragmatic for a home appliance.

### What's Protected

- All API endpoints require a Bearer token (except the first-run bootstrap flow).
- The Apex password lives only in the environment file (`/etc/symbiont/env`, mode 0400).
- The DuckDB and SQLite files are owned by the `symbiont` system user.
- Systemd service hardening: `PrivateTmp`, `NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths` limited to `/var/lib/symbiont`.

### What's Not Protected

- Token is stored as plaintext in SQLite. If the DB file is compromised, all tokens are exposed. Acceptable given physical security of the host.
- HTTP (not HTTPS) on the local network. Traffic between the browser and API is unencrypted on the LAN. Acceptable for local-only access. Tailscale encrypts traffic end-to-end for remote access.
- No rate limiting on API endpoints. Acceptable for single-user home use.

### Tailscale Configuration

Expose only port 8420 via Tailscale ACLs. Do not expose the DuckDB or SQLite files, SSH (unless separately configured), or any other service port.

```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["your-tailscale-devices"],
      "dst": ["symbiont-host:8420"]
    }
  ]
}
```
