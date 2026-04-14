# AI Agent + Skills for Symbiont

## Context

Symbiont already exposes a rich MCP surface (30 tools: parameters, livestock, journal, outlets, alerts, etc.) and stores structured facts about a tank (Tank Profile, Livestock, Alert Rules, Journal). Today, using Claude Desktop with Symbiont's MCP server gives you tool access but no opinionated persona and no repeatable workflows — every session starts cold.

The old ad‑hoc "Marine Biologist" prompt wired a persona, tank facts, dosing math, and safety guardrails into a single blob, all hand-tailored to one specific tank. We want the same value but:
- Generalized so any Symbiont user benefits.
- Personalized automatically from data already in the DB (tank volume, livestock, target parameters).
- User-tunable for tone + product preferences without hand-editing prompts.
- Delivered as portable **Claude Code skills** (markdown + frontmatter), not an in-app LLM.

**Outcome:** `symbiont skills install` drops a curated skill pack into `~/.claude/skills/symbiont/`. Claude Code / Claude Desktop auto-discovers them. Each skill, on invocation, calls a new MCP tool `get_agent_context` to pull live tank facts, then executes its workflow using existing MCP tools.

## Design

### 1. Skill pack (embedded markdown)

New package `internal/agent/` with `skills/` subdir embedded via `embed.FS`. Each skill = one folder with `SKILL.md` frontmatter:

```yaml
---
name: water-test-analysis
description: Analyze reef water test results (Alk/Ca/Mg/NO3/PO4) and produce a safe multi-day dosing plan. Use when the user pastes test numbers.
---
```

**Starter skills** (ship these):
- `water-test-analysis` — parse test values, compare to targets from tank profile, produce dosing schedule with safety limits (Mg first, ≤0.5 dKH/day, etc.).
- `weekly-maintenance` — generate a maintenance checklist pulling recent journal entries + alert events.
- `livestock-health-check` — walk through each livestock item, prompt for observations, log via `add_livestock_observation`.
- `new-addition-acclimation` — drip acclimation + QT guidance for a new fish/coral.
- `parameter-trend-review` — pull last 7/30 days of probe history, flag drift.
- `outlet-audit` — review outlet event log for anomalies.

Skill bodies reference `get_agent_context` MCP tool for tank-specific facts (volume, livestock, targets, persona, product preferences) instead of hardcoding them.

### 2. `get_agent_context` MCP tool

New tool in `internal/mcp/tools.go`. Returns a structured system-prompt block assembled from:
- Tank Profile (display + sump) — volume, dimensions, type
- Livestock inventory (fish / coral / invert with species)
- Active alert rules (reveals user's target parameters)
- **Agent settings** (new SQLite table, see §3) — persona tone, dosing product line, custom guardrails

Backed by a new API endpoint `GET /api/agent/context` so CLI and MCP both get it from the same place.

### 3. Agent settings (SQLite)

New single-row table `agent_settings`:
- `tone` — analytical | casual | terse (default analytical)
- `dosing_product_line` — brs_pharma | red_sea | tropic_marin | generic | none
- `net_volume_gallons` — optional override (otherwise derived from tank profile sum)
- `custom_guardrails` — free-text markdown appended to context
- `enabled_skills` — JSON array; skills not listed are omitted on install
- `updated_at`

Migrations in `internal/db/sqlite.go`. Queries in `internal/db/sqlite_queries.go` (`GetAgentSettings`, `UpsertAgentSettings`).

### 4. API endpoints (`internal/api/agent.go`)

- `GET /api/agent/settings` → current settings
- `PUT /api/agent/settings` → update
- `GET /api/agent/context` → assembled context markdown (used by MCP tool and CLI)
- `GET /api/agent/skills` → list of available skills with frontmatter metadata + enabled state

### 5. CLI (`cmd/symbiont`)

- `symbiont agent context` — print assembled context (debugging)
- `symbiont agent settings` — show settings; `--set tone=casual` etc.
- `symbiont skills list` — list built-in skills + enabled status
- `symbiont skills install [--dir ~/.claude/skills/symbiont]` — write enabled skills' markdown to target dir; idempotent (overwrite).
- `symbiont skills uninstall` — remove target dir.

Skills are served from the API (`GET /api/agent/skills/:name/body`) so a remote CLI install works the same as local. Fallback: if CLI is running on the same host as API with direct FS access, read from embed directly via a shared loader in `internal/agent/skills.go`.

### 6. Frontend: Settings > Agent tab

New tab added at `frontend/src/pages/Settings.tsx` (tabs array ~line 76). New page `frontend/src/pages/settings/Agent.tsx`:
- Persona controls (tone select, dosing product select, net volume override)
- Custom guardrails textarea
- Enabled skills — list with toggle + description (read from `/api/agent/skills`)
- "Preview context" panel showing assembled markdown
- Copy-paste install instructions: `symbiont skills install` + path

Design: follow `docs/design/DESIGN.md` — surface hierarchy, no 1px borders, rounded-2xl, etc.

## Files to create / modify

**New**
- `internal/agent/skills.go` — embed loader, frontmatter parser (reuse `yaml.v3`)
- `internal/agent/skills/*/SKILL.md` — starter skill pack (6 files)
- `internal/agent/context.go` — assembles context from tank profile + livestock + alerts + settings
- `internal/api/agent.go` — 4 handlers
- `internal/cli/agent.go`, `internal/cli/skills.go` — new CLI subcommands
- `frontend/src/pages/settings/Agent.tsx`
- `frontend/src/hooks/useAgent.ts` — TanStack Query hooks

**Modify**
- `internal/db/sqlite.go` — migration for `agent_settings`
- `internal/db/sqlite_queries.go` — Get/Upsert agent settings
- `internal/db/sqlite_models.go` — `AgentSettings` struct
- `internal/mcp/tools.go` — register `get_agent_context`, `list_skills`, `get_skill` tools
- `internal/api/router.go` (wherever routes live) — wire 4 new endpoints
- `frontend/src/pages/Settings.tsx` — add 'agent' tab
- `frontend/src/api/client.ts` — agent/skills client fns
- `docs/impl-*.md` — add Phase 8 (or append to 06/07) tracking this feature

## Verification

1. `go test ./...` — new unit tests for `internal/agent/skills.go` (frontmatter parse), `internal/agent/context.go` (context assembly with fixtures), and `internal/api/agent.go` (handler tests via `httptest`).
2. `go build ./...` and start poller + api + mcp.
3. Exercise API: `curl /api/agent/context` returns markdown with user's actual tank + livestock facts.
4. `symbiont agent settings --set tone=casual` then re-fetch context → tone change reflected.
5. `symbiont skills install --dir /tmp/sktest` → inspect files; confirm frontmatter + body.
6. Wire `/tmp/sktest` into Claude Code, paste fake test results, confirm `water-test-analysis` skill triggers and calls `get_agent_context` to pull real volume/livestock.
7. Frontend: open Settings > Agent, toggle a skill off, re-run install, confirm that skill is excluded.
8. MCP inspector (`npx @modelcontextprotocol/inspector`) — confirm `get_agent_context` appears and returns expected content.

## Phase 2 — Remote MCP (mobile access via claude.ai Projects)

Goal: expose Symbiont's MCP server over HTTPS so it can be added as a connector in a Claude.ai Project, giving the user tool access from the mobile app. The agent persona (already produced by `get_agent_context`) gets pasted into the Project's custom instructions.

### 2a. Token hardening (prerequisite — do first)

Current auth is a single shared bearer token with full privileges. Fine on LAN; risky when exposed publicly to claude.ai.

Changes to `tokens` table (`internal/db/sqlite.go` migration):
- `label` TEXT — human name ("claude-mobile", "cli-laptop")
- `scope` TEXT — `read` | `control` | `admin` (comma-sep if combining)
- `last_used_at` TIMESTAMP — updated by auth middleware
- `created_at` already exists

Middleware (`internal/api/middleware.go` or wherever auth lives):
- Parse token → look up row → enforce scope per route.
- Tag `context.Context` with token label; `initiated_by` in outlet event log becomes `"mcp:claude-mobile"` instead of bare `"mcp"`.

Route/tool scope matrix:
- `read`: all `get_*` MCP tools, all `GET` API routes
- `control`: `control_outlet`, `set_feed_mode`, `add_*`, `update_*`, `add_journal_entry`, alert rule CRUD
- `admin`: token management, tank profile edit, agent settings edit, backup

Frontend Settings > Tokens:
- Label + scope dropdown at creation time
- Show last-used timestamp, revoke button
- Default scope: `read` only — user must opt in to `control`

### 2b. HTTP MCP transport

Add Streamable HTTP transport to the MCP server (the `mcp-go` lib supports it). Wire it as a mounted handler on the existing API server at `POST /api/mcp` so it shares TLS + auth + reverse proxy. Standard bearer auth; scope enforced the same as REST.

Deployment doc (`docs/deployment-remote-mcp.md`): recommend Tailscale Funnel or Cloudflare Tunnel for public TLS without port forwarding.

### 2c. "Copy for claude.ai" UX

In Settings > Agent:
- Button: "Copy persona for claude.ai Project" → copies the output of `get_agent_context` to clipboard, ready to paste into a Project's custom instructions.
- Button: "Create connector token" → generates a `read`-scoped token labeled `claude-mobile` and shows the MCP URL + token once.
- Short inline walkthrough: Claude.ai → Project → Connectors → add URL + token → paste persona.

Skills aren't auto-loaded by claude.ai Projects, so the persona string should include a "Available workflows" section listing each skill's name + description, letting the user ask for them by name.

### 2d. Files to create / modify (phase 2)

**New**
- `internal/api/mcp_http.go` — HTTP MCP transport wiring
- `docs/deployment-remote-mcp.md`

**Modify**
- `internal/db/sqlite.go` — migration for token scopes/labels
- `internal/db/sqlite_queries.go` — token CRUD updates
- `internal/api/middleware.go` — scope enforcement, label propagation
- `internal/mcp/tools.go` — scope checks on write tools
- `frontend/src/pages/settings/Tokens.tsx` — label/scope UI
- `frontend/src/pages/settings/Agent.tsx` — clipboard + connector-token actions

## Deliberately out of scope

- No in-app LLM, no API key storage, no chat UI. Agent runs externally via Claude Code/Desktop/claude.ai.
- No user-authored skills in v1.
- No OAuth 2.1 / DCR for remote MCP — static bearer is sufficient for single-user.
- No MCP "prompts" primitive yet (inconsistent client support).
- No mTLS, no WAF. Rely on Tailscale/Cloudflare Tunnel for edge protection.

## Commit cadence

Per the "commit incrementally" preference:

**Phase 1 (skills + persona, desktop):**
1. DB migration + agent settings queries + models
2. `internal/agent/` (skills loader + context assembly + starter skill pack)
3. API endpoints + MCP tools
4. CLI subcommands
5. Frontend Agent settings tab

**Phase 2 (remote MCP for mobile):**
6. Token scopes/labels migration + middleware + audit enrichment
7. Tokens UI updates (scope, label, last-used)
8. HTTP MCP transport + deployment doc
9. "Copy for claude.ai" + connector-token flow in Agent settings
