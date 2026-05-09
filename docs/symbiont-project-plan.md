# Symbiont — Project Plan
> A local-first Neptune Apex dashboard and data platform

> Last updated: May 2026 — All phases 1–7 complete.

---

## Vision

Replace Apex Fusion with a self-hosted stack that collects, stores, and visualizes all your aquarium data locally — with full outlet control, AI integration, and no dependency on Neptune's cloud.

**Design principles:**
- Fully local, no cloud dependency
- Runs on the existing NixOS mini PC
- Go backend, TypeScript/React frontend
- DuckDB for time-series telemetry (embedded, no server process)
- SQLite for relational app state and configuration
- Clean REST/JSON API decoupled from Apex's quirky auth
- AI-first platform — MCP server and CLI as first-class integration surfaces
- "Abyssal Laboratory" dark-mode UI with glassmorphism and ambient glow aesthetics

---

## What We Know About the Apex (AOS 5+)

### Local API

| Endpoint | Method | Description |
|---|---|---|
| `/rest/login` | POST | Authenticate, get session cookie |
| `/rest/status` | GET | Full JSON: probes, outputs, controller info |
| `/rest/outlets` | GET | Outlet states only |
| `/rest/outlets/{id}` | PUT | Control a single outlet |

### Auth

Session-cookie auth: POST credentials to `/rest/login`, server sets `connect.sid` session cookie, all subsequent requests include it. The Apex client automatically re-authenticates on 401.

---

## Architecture

Four Go binaries + React frontend, all on one NixOS host. See `docs/symbiont-architecture.md` for the full technical breakdown.

### Key Design Decisions

**Go throughout the backend.** Single binaries per service, low memory footprint, stable for long-running processes.

**REST/JSON, not gRPC.** All communication between services is HTTP/JSON. Trivially debuggable with curl.

**Single writer to DuckDB.** The Poller is the sole writer. The API server is read-only against DuckDB. Never the reverse.

**DuckDB vs SQLite — clear division.**
- DuckDB: tank telemetry — append-only time-series, write-heavy, columnar
- SQLite: app state — relational config, user-facing state, rarely written

**API proxies all Apex communication.** The frontend and MCP server never talk to the Apex directly.

**MCP and CLI are API clients.** Nothing is duplicated. Both wrap the REST API.

---

## Database Design

### DuckDB — Time-Series Telemetry

Four tables: `probe_readings`, `outlet_states`, `power_events`, `controller_meta`. All append-only, written by the Poller every 10 seconds.

### SQLite — App State and Configuration

Over 20 tables covering: auth tokens, devices, probe/outlet config, alert rules, alert events, notifications, backups, dashboard items, measurement parameters, measurements, livestock, livestock observations, tank profile, journal entries, agent settings, system events, dosing products/schedules/logs, maintenance tasks/logs, outlet programs.

See `docs/symbiont-architecture.md` for the full schema.

---

## Project Structure

```
symbiont/
├── cmd/
│   ├── poller/main.go
│   ├── api/main.go
│   ├── mcp/main.go
│   └── symbiont/main.go
│
├── internal/
│   ├── agent/               # Agent context assembly and skills
│   ├── alerts/              # Alert evaluation engine
│   ├── apex/                # Apex HTTP client (auth, session, models)
│   ├── api/                 # HTTP handlers, middleware, SSE broadcaster
│   ├── audit/               # Audit log helpers
│   ├── backup/              # Backup and retention logic
│   ├── cli/                 # CLI commands and output formatting
│   ├── config/              # Config loading from env
│   ├── db/                  # DuckDB + SQLite packages (many files)
│   ├── enums/               # Shared vocabulary sets for validation
│   ├── events/              # Internal event bus
│   ├── journal/             # Journal auto-logging from event bus
│   ├── kits/                # Test kit definitions
│   ├── mcp/                 # MCP tool implementations (14 files)
│   ├── notify/              # Notification delivery (ntfy.sh)
│   └── poller/              # Polling loop
│
├── frontend/
│   └── src/
│       ├── api/             # Typed fetch client (22 files + barrel)
│       ├── components/      # Reusable components
│       ├── hooks/           # TanStack Query hooks + SSE hook
│       └── pages/           # Route-level pages + settings tabs
│
├── testdata/                # Apex API response fixtures (JSON)
├── docs/                    # Architecture docs
├── flake.nix
├── go.mod
└── .env.example
```

---

## Go Dependencies

```
# Core
net/http                        # stdlib HTTP server — no framework needed
github.com/marcboeker/go-duckdb # DuckDB driver
modernc.org/sqlite              # SQLite driver (pure Go, no cgo)
github.com/joho/godotenv        # .env file loading

# CLI
github.com/spf13/cobra          # CLI framework with subcommands

# MCP
github.com/mark3labs/mcp-go     # MCP server implementation

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
    "@tanstack/react-query": "^5",
    "@dnd-kit/core": "^6",
    "@dnd-kit/sortable": "^8",
    "lucide-react": "latest",
    "clsx": "latest",
    "tailwind-merge": "latest"
  },
  "devDependencies": {
    "typescript": "^5",
    "vite": "^5",
    "tailwindcss": "^3"
  }
}
```

No Tremor, no shadcn/ui. All UI is hand-built with Tailwind following the Abyssal Laboratory design system in `docs/design/DESIGN.md`.

---

## API Design

See full route table in `docs/symbiont-architecture.md`. Summary:

### Authentication

All requests require `Authorization: Bearer <token>`. Token scopes: `read` | `write` | `control` | `admin`. Only `admin` tokens can manage other tokens or change config.

### Key Endpoints

```
GET  /api/probes                  → current probe readings
GET  /api/probes/{name}/history   → time-series (DuckDB bucketed)
GET  /api/outlets                 → current outlet states
PUT  /api/outlets/{id}            → outlet control [control scope]
GET  /api/feed                    → feed mode status
PUT  /api/feed                    → set feed mode [control scope]
GET  /api/dashboard               → customizable dashboard items
GET  /api/measurements            → manual water chemistry log
GET  /api/livestock               → livestock inventory
GET  /api/journal                 → journal entries
GET  /api/dosing/schedules        → dosing schedules + due dates
GET  /api/maintenance/tasks       → maintenance task definitions
GET  /api/tasks/due               → unified due queue (dosing + tasks)
GET  /api/agent/context           → AI context bundle for MCP
GET  /api/stream                  → SSE real-time push
```

---

## MCP Server

30+ tools covering the full feature surface. All tools are thin clients over the REST API — no direct DB access. Supports two transports:

- **stdio** — for local Claude Desktop / Claude Code
- **HTTP/SSE** — for claude.ai remote connections via Tailscale (see `docs/deployment-remote-mcp.md`)

---

## CLI

Single binary `symbiont` with subcommands for all major features. All commands support `--json` flag for scripting.

```
symbiont probes current
symbiont outlets set <id> <ON|OFF|AUTO>
symbiont measurements add
symbiont dosing log
symbiont livestock list
symbiont journal add
symbiont agent context
symbiont system status
symbiont auth tokens create --label "claude-desktop" --scope control
```

---

## Auth

Four scopes:
- **read** — read-only access, no data modification, no hardware control
- **write** — data management (measurements, livestock, dosing, maintenance, journal, dashboard) but no hardware control
- **control** — write + outlet on/off/auto + feed mode
- **admin** — full access including config, tokens, backup, agent settings

Tokens are 64-char hex strings stored in SQLite. First token is auto-generated on first run.

---

## Alerting

Rules stored in SQLite, evaluated by background goroutine in the API server on each poll cycle. Three rule types: above threshold, below threshold, outside range. Notifications delivered via ntfy.sh. Cooldown period prevents notification spam.

---

## Backup Strategy

Nightly systemd timer copies both DuckDB and SQLite files to a configurable backup directory. Keeps last N backups (configurable). Status visible in Settings > Backup.

---

## NixOS Integration

Four systemd services + two systemd timers (backup + cleanup). Services run as `symbiont` system user with read-write access limited to `/var/lib/symbiont`. Config loaded from `/etc/symbiont/env`.

---

## Completed Phases

### Phase 1 — Data Collection ✓
Apex client, DuckDB schema, polling loop, systemd service.

### Phase 2 — API Server ✓
Full REST API, SQLite schema, auth middleware, history endpoint, outlet control, SSE.

### Phase 3 — CLI ✓
Cobra CLI with all subcommands and JSON output mode.

### Phase 4 — Frontend MVP ✓
Vite + React + TypeScript + Tailwind, full dashboard with real-time SSE updates.

### Phase 5 — MCP Server and AI Layer ✓
30+ MCP tools, stdio + HTTP transport, Claude Desktop + claude.ai integration. Agent settings, embedded skills pack, agent context assembly.

### Phase 6 — Alerts, Notifications, and Polish ✓
Alert engine with debounce, ntfy.sh, CSV export, backup automation, dosing schedules, maintenance tasks, livestock tracking, water chemistry measurements, journal, tank profile, devices, dashboard customization.

### Phase 7 — Dashboard Layout Builder ✓
Customizable dashboard via `dashboard_items` SQLite table. Settings > Dashboard tab with drag-and-drop reordering (dnd-kit). Six item types: probe, outlet, device, separator, feed_mode, measurement. Display mode (normal/compact) per item.

> Note: The original Phase 7 plan described a React Flow canvas with draggable nodes. The actual implementation used a simpler, more practical approach: a sortable item list in Settings that customizes what appears on the main dashboard — same goal (customization), different implementation (list vs. canvas).
