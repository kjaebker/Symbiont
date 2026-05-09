# Symbiont — Implementation Plan
> Index and conventions for the phased task list

---

## Files

| File | Phase | Description |
|---|---|---|
| `impl-01-data-collection.md` | Phase 1 | Repo setup, Apex client, Poller, DuckDB |
| `impl-02-api-server.md` | Phase 2 | API server, SQLite, auth, SSE |
| `impl-03-cli.md` | Phase 3 | CLI binary with all subcommands |
| `impl-04-frontend.md` | Phase 4 | React frontend MVP |
| `impl-05-mcp.md` | Phase 5 | MCP server and AI integration |
| `impl-06-polish.md` | Phase 6 | Alerts, notifications, backup, export |
| `impl-07-layout-builder.md` | Phase 7 | Dashboard layout builder (customizable items, drag-and-drop) |

---

## Conventions

### Task Status Markers

```
[ ]   Not started
[~]   In progress
[x]   Complete
[!]   Blocked or needs decision
```

### Task Types

Tasks are labeled by type where relevant:

- **[code]** — write or modify code
- **[config]** — configuration or environment
- **[test]** — write or run a test
- **[verify]** — manual verification step
- **[decision]** — needs a choice before proceeding
- **[research]** — needs investigation before implementation

### Dependency Notation

Tasks marked `↳ depends on: [X]` must not be started until the referenced task is complete.

---

## Development Environment

Before starting Phase 1, the dev environment should be confirmed:

- [x] [config] NixOS mini PC is the primary workstation
- [x] [config] `flake.nix` in repo root provides Go toolchain, DuckDB CLI, SQLite CLI
- [x] [config] `.env` file in project root for local dev (not committed)
- [x] [config] VS Code or editor with Go plugin configured
- [x] [verify] `go version` → 1.23+ (1.25.0 via nix)
- [x] [verify] Apex is reachable at its local IP from the mini PC
- [x] [verify] Browser DevTools capture plan is understood (see Phase 1)

---

## Phase Sequencing

Phases 1–3 are strictly sequential. Phase 4 (frontend) can begin after Phase 2 API endpoints are functional — the CLI (Phase 3) can be developed in parallel with Phase 4. Phase 5 (MCP) requires Phase 3 (CLI) to be working. Phases 6 and 7 are independent of each other and can proceed in any order after Phase 4.

```
Phase 1 → Phase 2 → Phase 3 ─────────────────────────→ Phase 5
                   └──────→ Phase 4 → Phase 6 → Phase 7
```

---

## Repository Layout (current)

```
symbiont/
├── cmd/
│   ├── poller/main.go
│   ├── api/main.go
│   ├── mcp/main.go
│   └── symbiont/main.go
├── internal/
│   ├── agent/               # Agent context assembly and skills pack
│   ├── alerts/              # Alert evaluation engine
│   ├── apex/                # Apex HTTP client (auth, session, models)
│   ├── api/                 # HTTP handlers, middleware, SSE broadcaster
│   ├── audit/               # Audit log helpers
│   ├── backup/              # Backup and retention logic
│   ├── cli/                 # CLI commands and output formatting
│   ├── config/              # Config loading from env
│   ├── db/                  # DuckDB + SQLite packages (many files each)
│   ├── enums/               # Shared enum/vocabulary sets (validated against schema)
│   ├── events/              # Internal event bus (publish/subscribe)
│   ├── journal/             # Journal auto-logging driven by event bus
│   ├── kits/                # Test kit definitions for chemistry measurements
│   ├── mcp/                 # MCP tool implementations (14 files, 30+ tools)
│   ├── notify/              # Notification delivery (ntfy.sh)
│   └── poller/              # Polling loop
├── frontend/
│   └── src/
│       ├── api/             # Typed fetch client (22 files + barrel index)
│       ├── components/      # Reusable components (CardBase, charts, etc.)
│       ├── hooks/           # TanStack Query hooks + SSE hook
│       └── pages/           # Route pages + settings tabs
├── testdata/                # Apex API response fixtures (JSON)
├── docs/                    # Architecture docs + design system
├── flake.nix
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

---

## Definition of Done (per phase)

Each phase has a **Deliverable** — a concrete, verifiable end state. A phase is not complete until its deliverable is met, regardless of individual task checkboxes.

Deliverables are listed at the top of each phase file.
