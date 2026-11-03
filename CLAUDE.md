# CLAUDE.md — Symbiont

This file is the first thing you should read in every session. It tells you what this project is, how it's structured, and the conventions to follow without being told.

---

## What This Project Is

Symbiont is a local-first Neptune Apex aquarium controller dashboard. It replaces Apex Fusion (Neptune's cloud dashboard) with a self-hosted stack running on a NixOS mini PC. It polls a Neptune Apex controller over the local network, stores time-series data, and serves a React dashboard.

The codebase is Go (backend) and TypeScript/React (frontend). There is no cloud. There is no external API dependency beyond the Apex hardware on the local network.

---

## Repository Structure

```
symbiont/
├── cmd/
│   ├── symbiont/            # Production binary: `symbiont serve` (poller + API + alerts in one process) and the CLI
│   ├── api/main.go          # Dev server: same in-process stack, frontend served from disk
│   ├── mcp/main.go          # MCP server (stdio) for AI assistant integration
│   └── poller/main.go       # Standalone poller (legacy; cannot run alongside serve: DuckDB file lock)
├── internal/
│   ├── apex/                # Apex HTTP client (auth, session, models)
│   ├── db/                  # DuckDB and SQLite packages (separate files)
│   ├── poller/              # Polling loop
│   ├── api/                 # HTTP handlers, middleware, SSE broadcaster
│   ├── cli/                 # CLI commands and output formatting
│   ├── mcp/                 # MCP tool implementations
│   ├── alerts/              # Alert evaluation engine
│   ├── notify/              # Notification delivery (ntfy.sh)
│   ├── backup/              # Backup and retention logic
│   └── config/              # Config loading from env
├── frontend/                # Vite + React + TypeScript
│   └── src/
│       ├── api/             # Typed fetch client
│       ├── components/      # Reusable components
│       ├── hooks/           # TanStack Query hooks + SSE hook
│       └── pages/           # Route-level page components
├── testdata/                # Apex API response fixtures (JSON)
├── docs/                    # Architecture docs, API notes
├── go.mod
└── .env                     # Local dev secrets (not committed)
```

All shared logic lives in `internal/`. There are no public packages — this is not a library. Production runs a single process (`symbiont serve`, systemd unit `symbiont`); release builds embed `frontend/dist`.

---

## Build Commands

```bash
# Build all binaries
go build ./...

# Build individual binaries
go build ./cmd/symbiont
go build ./cmd/api
go build ./cmd/mcp

# Release build with embedded frontend
make build

# Run the full stack locally with .env (load vars first, then run)
set -a && source .env && set +a
go run ./cmd/symbiont serve

# Release: checks, tag, push, wait for CI (then deploy to the server)
make release V=x.y.z

# Frontend
cd frontend && npm run dev     # Dev server (proxies /api to localhost:8420)
cd frontend && npm run build   # Production build to frontend/dist/
cd frontend && npx tsc --noEmit  # Type check without building
```

---

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/apex/...
go test ./internal/db/...
go test ./internal/api/...

# Run with verbose output
go test -v ./internal/apex/...

# Run with race detector (use for concurrency-sensitive packages)
go test -race ./internal/api/...

# Run a specific test
go test -run TestApexClientReauth ./internal/apex/...

# Frontend tests (vitest + Testing Library, jsdom)
cd frontend && npm test
```

All tests must pass before considering any task complete. If a test is failing and you can't fix it, say so — don't skip or comment it out.

---

## Code Conventions

### Go

**Error handling:**
- Always wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Never swallow errors silently
- API handlers return JSON errors via `writeError(w, status, message, code)`
- Poller and background goroutines log errors and continue — they never crash on external failures

**Logging:**
- Use `log/slog` exclusively — no `fmt.Println`, no `log.Printf`
- JSON handler in all binaries
- Always include structured fields, not string interpolation:
  - ✅ `slog.Error("poll failed", "err", err, "duration_ms", elapsed.Milliseconds())`
  - ❌ `slog.Error(fmt.Sprintf("poll failed: %v", err))`

**Database:**
- Raw SQL only — no ORM, no query builder
- DuckDB: `internal/db/duckdb.go` for writes (Poller), `internal/db/queries.go` for reads (API). Filter time-series reads by a `ts` bound passed as a parameter so zone maps can prune; avoid unbounded `MAX(ts)` / `ORDER BY ts DESC LIMIT 1` over whole tables
- SQLite: `internal/db/sqlite.go` and `internal/db/sqlite_queries.go`
- All queries are named functions with typed return values — no inline SQL in handlers
- DuckDB writes are wrapped in transactions via `WritePollCycle`
- Never join DuckDB and SQLite — combine at the application layer in handlers

**HTTP:**
- stdlib `net/http` only — no Gin, no Chi, no Echo
- All responses via `writeJSON(w, status, v)` helper
- All errors via `writeError(w, status, message, code)` helper
- Request bodies decoded via `readJSON(r, &v)` helper with size limit

**Concurrency:**
- Apex client session cookie protected by `sync.Mutex`
- SSE broadcaster uses `sync.RWMutex`
- Alert engine state map protected by `sync.Mutex`
- Use `context.Context` for cancellation — always propagate context to downstream calls

**Naming:**
- Exported types use full names: `ProbeReading`, `OutletState`, `AlertRule`
- Internal helpers use short names consistent with Go idioms
- Test files: `foo_test.go` alongside `foo.go` in the same package

**Testing:**
- Use `httptest.NewServer` for API handler tests — real HTTP, not mocks of the handler
- Use temp files for DuckDB tests, `:memory:` for SQLite tests
- Always clean up with `t.Cleanup(func() { ... })`
- Test table-driven where multiple cases exist

### TypeScript / React

**Fetching:**
- All API calls go through `src/api/client.ts` — never fetch directly in components
- TanStack Query for all server state — no `useState` + `useEffect` for data fetching
- SSE invalidates TanStack Query caches — components do not poll

**Components:**
- Functional components only, no class components
- Props interfaces defined inline above the component
- No prop drilling beyond 2 levels — use query hooks directly in leaf components

**Styling:**
- Tailwind utility classes only — no inline styles, no CSS modules
- Dark mode via `dark:` prefix — dark is the default (`<html class="dark">`)
- Never hardcode colors — use Tailwind's semantic palette
- **Follow the design system in `docs/design/DESIGN.md`** — all frontend work must conform to the "Abyssal Laboratory" creative direction:
  - Surfaces: use the layered surface hierarchy (`surface` → `surface-container-*`) for depth — no flat layouts
  - **No 1px borders.** Define boundaries through background color shifts or ghost borders (`outline-variant` at 15% opacity)
  - **No divider lines.** Use spacing or subtle surface shifts to separate items
  - Colors: Primary (cyan `#3adffa`), Secondary (green `#6dfe9c`), Tertiary (coral `#ff8796`) — never use pure white (#FFFFFF)
  - Use glassmorphism (`glass` utility) for floating elements
  - All transitions must be 300ms ease-in-out minimum (`transition-fluid`)
  - Roundedness: `rounded-2xl` (2rem) for cards, `rounded-full` for chips/pills
  - Shadows: use ambient glow (`shadow-glow-*`) not hard drop shadows
  - Typography: Manrope font, data values are hero elements (display size), labels are small uppercase with wide tracking
  - Reference the mockups in `docs/design/*.png` for layout guidance on each page

**State:**
- Server state: TanStack Query
- UI state: `useState` in the component that owns it
- No global state store (no Zustand, no Redux, no Context for data)

**Imports:**
- Use `@/` path alias for `src/` imports: `import { ProbeCard } from '@/components/ProbeCard'`

---

## Two Databases — Know Which Is Which

This is the most important architectural fact:

| Store | File | Owns | Who Writes |
|---|---|---|---|
| DuckDB | `telemetry.db` | All time-series data (probes, outlets, power events) | Poller only |
| SQLite | `app.db` | App state (config, tokens, alerts, event log) | API layer only |

**One process owns both databases.** `symbiont serve` opens DuckDB read-write once and shares the `*db.DuckDB` handle between the poller goroutine, the API handlers, the SSE ticker and the alert engine. DuckDB takes an exclusive file lock, so no other process (including `cmd/poller`) can open `telemetry.db` while the server runs.

**DuckDB is append-only and the Poller is the sole writer.** API handlers, SSE and alerts only read, even though the handle is read-write. Never write to DuckDB from the API layer. (Retention cleanup, `DeleteOldRows`, is the one deliberate exception.)

**"Current state" reads depend on the shared handle.** `CurrentProbeReadings`, `CurrentOutletStates`, `ControllerMeta` and `LastPollTime` use a per-table latest-`ts` value cached in memory as a lower bound, which lets DuckDB prune old row groups (a full `MAX(ts)` scan took ~1s in production). Writes through `WritePollCycle`/`Write*` advance it. A stale value is still a valid lower bound, so correctness holds; it is only fast if writes go through the same handle. Don't add a second DuckDB handle or an out-of-process writer without revisiting this.

**SQLite is the app's brain.** Config, tokens, alert rules, outlet event log, backup records. The Poller never touches SQLite.

When adding a new feature, the first question is: does this data belong in DuckDB (time-series, written by poller) or SQLite (app state, written by API)?

---

## Components and Their Responsibilities

| Component | Runs in | Talks To | Never Touches |
|---|---|---|---|
| Poller (`internal/poller`) | `symbiont serve` | Apex (HTTP), DuckDB (write) | SQLite, frontend |
| API (`internal/api`) | `symbiont serve` | DuckDB (read), SQLite (r/w), Apex (outlet control only) | DuckDB writes |
| Alert engine (`internal/alerts`) | `symbiont serve` | DuckDB (read), SQLite, notifiers | DuckDB writes |
| MCP (`cmd/mcp`, `/api/mcp`) | own process / API | API server (HTTP client) | Databases directly |
| CLI (`symbiont <cmd>`) | user shell | API server (HTTP client) | Databases directly |

The MCP server and CLI subcommands are thin HTTP clients over the REST API. They never touch the databases. The only database code under `cmd/` is the wiring in `cmd/symbiont/serve.go` (and its dev twin `cmd/api/main.go`); if you're writing queries in `cmd/`, something is wrong.

---

## API Conventions

**All responses are JSON.**

Success:
```json
{ "probes": [...] }
```

Error:
```json
{ "error": "human-readable message", "code": "machine_readable_code" }
```

**Auth:** Bearer token in `Authorization` header on every request. SSE endpoint uses `?token=` query param instead (browser EventSource limitation).

**The `initiated_by` field on outlet changes** must be set correctly: `"ui"`, `"cli"`, `"mcp"`, or `"api"`. This comes from the API handler context, not the caller.

---

## What NOT To Do

- **Do not use an ORM.** Raw SQL only. If you find yourself reaching for GORM or sqlx, stop.
- **Do not add a framework to the HTTP server.** stdlib `net/http` only.
- **Do not write to DuckDB from the API layer.** Only the poller writes; API handlers, SSE and alerts read through the shared handle.
- **Do not access databases from the MCP server or CLI.** They are HTTP clients only.
- **Do not use `fmt.Println` or `log.Printf` for logging.** Use `slog` everywhere.
- **Do not swallow errors.** Log and surface them.
- **Do not use `interface{}` or `any` loosely.** Keep types tight.
- **Do not add npm packages without checking if the functionality exists in the already-installed libraries** (lucide-react, uPlot, TanStack Query, tailwind-merge, clsx, dnd-kit).
- **Do not use localStorage or sessionStorage in React components.** The only localStorage usage is the auth token in `src/api/client.ts`.
- **Do not use `useEffect` for data fetching.** Use TanStack Query hooks.
- **Do not inline SQL in HTTP handlers.** All queries are named functions in the db package.
- **Do not use 1px solid borders, divider lines, or hard drop shadows in the frontend.** Follow the design system in `docs/design/DESIGN.md`.

---

## Environment Variables

All config comes from environment variables. In development, loaded from `.env` via `godotenv`. In production, from `/etc/symbiont/env` via systemd `EnvironmentFile`.

Key variables (see `.env.example` for full list):
```
SYMBIONT_APEX_URL      # e.g. http://192.168.1.100
SYMBIONT_APEX_USER     # Apex login username
SYMBIONT_APEX_PASS     # Apex login password
SYMBIONT_DB_PATH       # DuckDB file path
SYMBIONT_SQLITE_PATH   # SQLite file path
SYMBIONT_API_PORT      # Default: 8420
SYMBIONT_TOKEN         # Used by CLI and MCP server
```

---

## Implementation Plan

Work is organized into 7 phases. The phase files are in the repo at:

```
docs/impl-00-overview.md
docs/impl-01-data-collection.md
docs/impl-02-api-server.md
docs/impl-03-cli.md
docs/impl-04-frontend.md
docs/impl-05-mcp.md
docs/impl-06-polish.md
docs/impl-07-layout-builder.md
```

Always check which phase is active before starting work. Don't skip ahead — later phases depend on earlier ones being solid.

When a task is complete, mark it `[x]` in the relevant phase file.

---

## Current Status

> Update this section at the start of each session.

**Active phase:** Phase 7 — Layout Builder (phases 1–6 substantially complete)
**Last completed task:** Agent + Skills feature (feat/agent-skills) — full two-phase implementation: embedded skills pack, agent context assembly, MCP tools, CLI subcommands, Settings > Agent tab, token scopes/middleware, HTTP MCP transport, claude.ai connector flow
**Next task:** Phase 7 tasks (or merge feat/agent-skills → main first)
**Blockers:** None
