# Architecture Refactor Roadmap

A multi-phase refactor of Symbiont kicked off from an architecture review on 2026-05-06. This document records what shipped, what's left, and concrete starting points for picking the work back up.

---

## What shipped

| Phase | PR | What changed |
|---|---|---|
| 1 | [#59](https://github.com/kjaebker/Symbiont/pull/59) | File splits: `sqlite_queries.go` (2540→16 files), `mcp/tools.go` (2012→13 files), `client.ts` (954→22 files + barrel), `Settings.tsx` (3084→11 tabs + shared). |
| 2 | [#60](https://github.com/kjaebker/Symbiont/pull/60) | Pattern extractions: shared image-handler helper (`livestock.go` 881→401), `internal/enums` package (11 validation maps consolidated), typed `qk` query-key factory (19 hooks + 4 components). |
| 3 | [#61](https://github.com/kjaebker/Symbiont/pull/61) | Schema v1 baseline + migration runner (drops 3 table-recreation funcs, all ALTERs inlined, `schema_versions` table); v2 partial index on active alert events; `ListDevices` N+1 → 3 queries; debounced batched `TouchToken`; v3 promoting `events.initiated_by` from JSON to indexed column. |

---

## What's left, ordered by ROI

### Tier A — clear wins, modest effort

#### A1. Schema-drift test between `internal/enums` and SQLite CHECK constraints
**Why:** Phase 2 consolidated enum vocabularies into `internal/enums/enums.go`, but each `Set` is still hand-kept in sync with the matching CHECK constraint in `internal/db/sqlite_schema.go`. A test that parses the schema once and asserts no drift would catch a real class of bug.
**Effort:** small (half-day).
**Where to start:** new `internal/enums/drift_test.go`. Open a test SQLite, query `SELECT sql FROM sqlite_master WHERE type='table'`, regex out the `CHECK(col IN (...))` literals, compare against the matching `enums.X.Values()`. Skip enums without a 1:1 CHECK (e.g. `InitiatedBy`, `HistoryIntervals`, `ImageExtensions` — comment them out of the comparison map).

#### A2. Card component base extraction
**Why:** 9 card variants — `LivestockCard`, `MeasurementCard`, `ProbeCard`, `OutletCard`, `DeviceCard`, `CompactCard`, `PowerPairCard`, `FeedCard`, `DailyPromptCard`. Each was built from scratch. Extract a base `Card` with composition slots before the 10th lands.
**Effort:** medium (1 day).
**Where to start:** look at the three biggest first (`DeviceCard.tsx` 363 lines, `ProbeCard.tsx` 207 lines, `LivestockCard.tsx` 93 lines). The repeated pattern is glass surface + status indicator strip + header (icon + name + meta) + body slot. A `<Card status={…} icon={…} title={…} meta={…}>{body}</Card>` covers most variants.

#### A3. Frontend bundle code-splitting
**Why:** `npm run build` warns that `dist/assets/index-*.js` is 744 KB. Settings + LivestockDetail + Journal are heavy and not on the critical path.
**Effort:** small (couple of hours).
**Where to start:** convert `src/App.tsx` route definitions to `React.lazy()` with `Suspense`. Settings (3 KB → multiple chunks for each tab), Livestock detail, Journal, History are obvious candidates.

#### A4. Decide the fate of `outlet_program_history`
**Why:** The DB review flagged this table as write-only — never SELECTed in this codebase. Either it's planned future work, or it's dead.
**Effort:** small (30 min investigate; either drop with a v4 migration or wire up a UI later).
**Where to start:** `grep -r "outlet_program_history" .` — confirm no consumers. Then either `DROP TABLE` as v4, or add a TODO with intended use.

---

### Tier B — bigger, higher-value architectural pieces

#### B1. Declarative scope rules
**Why:** `internal/api/middleware.go:isAdminRoute` is a hardcoded URL switch. Every new admin-only route needs two edits: register the route AND update the middleware. Drift waiting to happen.
**Effort:** medium (1 day).
**Where to start:** introduce a small registration wrapper, e.g. `s.adminMux.HandleFunc(...)` and `s.controlMux.HandleFunc(...)`, that records the required scope per pattern. The `Auth` middleware looks up the scope by matched pattern instead of regex-matching the URL. Alternative: per-handler decorator (`requireScope("admin")(s.HandleFoo)`) — more verbose but no central registry.

#### B2. Form abstraction
**Why:** `AlertRuleForm.tsx`, `LivestockForm.tsx`, the inline forms inside Settings tabs, the dosing/maintenance forms — they all do the same thing: validated input fields + save/cancel + error display. There's no shared `<Form>` or `<Field>` primitive.
**Effort:** medium (1–2 days).
**Where to start:** spec out a tiny `Field` API (label + input + error) and a `Form` that wires it to a TanStack mutation. Consider adopting `react-hook-form` if it earns its weight, but a hand-rolled minimal version may be enough for this codebase.

#### B3. Settings sub-page route refactor
**Why:** Phase 1 split Settings.tsx into `pages/settings/<Tab>Tab.tsx`, but they're still all rendered conditionally in one `<Settings>` component. Each tab could be a real route (`/settings/dashboard`, `/settings/probes`, etc.) so deep-linking works and only one tab's code loads at a time. Stacks naturally with A3 (code-splitting).
**Effort:** small (half-day).
**Where to start:** `src/App.tsx` route definitions; replace `<Route path="/settings" element={<Settings />} />` with a nested route tree.

#### B4. Larger pages: extract sub-sections
**Why:** Phase 1 left `LivestockDetail.tsx` (835), `Journal.tsx` (707), `Chemistry.tsx` (624) intact. Each likely has cohesive sub-sections that should be extracted, but lower priority than Settings was.
**Effort:** medium each (1 day per page).
**Where to start:** read each file, look for top-level sections (often introduced by `// === Section ===` banners or by visual grouping in the JSX). Extract to `pages/<page>/<Section>.tsx`. Same approach as Settings split.

---

### Tier C — investigations / decisions

#### C1. SSE double-path investigation
**Why:** `internal/api/sse.go` runs *two* paths to clients: a 10-second poll of DuckDB (`StartSSEPoller`) AND bus-event subscriber (`RegisterSSESubscriber`). `EvtPollCycleCompleted` is published but not consumed by SSE. Either enrich the event with full state and remove the poller, or document why both exist.
**Effort:** small to investigate, medium to unify.
**Where to start:** `git log internal/api/sse.go` to read the history; understand whether the polling is for "no clients connected" backpressure or genuinely needed.

#### C2. Unify dosing schedules + maintenance tasks behind a `recurring_task` abstraction
**Why:** Both have nearly identical schemas (`frequency`, `interval_days`, `enabled`, `next_due_at`, `last_completed_at`) and parallel handlers/CLI/MCP. Currently flagged as **probably skip** — different UX, small duplication.
**Effort:** large.
**Decision:** revisit only if a third recurring-task type lands.

---

### Tier D — large, defer until needed

#### D1. OpenAPI / generated clients
**Why:** MCP, CLI, and Frontend each hand-code validation, types, and request shapes for the same HTTP API. An OpenAPI spec generated from Go code (or vice versa) would remove a class of drift bugs.
**Effort:** large (1+ week).
**Trigger to revisit:** when a client/server type mismatch causes a real bug, or when the API surface grows large enough that drift becomes routine.
**Where to start:** evaluate `oapi-codegen` (Go side) + `openapi-typescript` (TS side). Generate the spec from handler signatures or write it by hand once and have both sides consume.

---

## Stacking strategy

Phases 1–3 are stacked PRs. Land them bottom-up: #59, then #60 (auto-retargets to main), then #61.

For new work, start a fresh branch off main once the existing stack lands. Don't try to add to an open phase PR — keep them mechanical and small.

## Branch naming

- `refactor/<topic>` for cross-cutting refactors
- `perf/<topic>` for targeted perf fixes
- `feat/<topic>` for new features

The phase-based names (`refactor/phase-1-splits` etc.) were a one-time convenience for the architecture-review work.
