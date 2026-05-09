# Architecture Refactor Roadmap

A multi-phase refactor of Symbiont kicked off from an architecture review on 2026-05-06. This document records what shipped, what's left, and concrete starting points for picking the work back up.

---

## What shipped

| Phase | PR | What changed |
|---|---|---|
| 1 | [#59](https://github.com/kjaebker/Symbiont/pull/59) | File splits: `sqlite_queries.go` (2540→16 files), `mcp/tools.go` (2012→13 files), `client.ts` (954→22 files + barrel), `Settings.tsx` (3084→11 tabs + shared). |
| 2 | [#60](https://github.com/kjaebker/Symbiont/pull/60) | Pattern extractions: shared image-handler helper (`livestock.go` 881→401), `internal/enums` package (11 validation maps consolidated), typed `qk` query-key factory (19 hooks + 4 components). |
| 3 | [#61](https://github.com/kjaebker/Symbiont/pull/61) | Schema v1 baseline + migration runner (drops 3 table-recreation funcs, all ALTERs inlined, `schema_versions` table); v2 partial index on active alert events; `ListDevices` N+1 → 3 queries; debounced batched `TouchToken`; v3 promoting `events.initiated_by` from JSON to indexed column. |
| A1 | [#62](https://github.com/kjaebker/Symbiont/pull/62) | Schema-drift test: `internal/enums/drift_test.go` asserts no drift between `enums.Set` values and SQLite CHECK constraints. |
| A2 | [#63](https://github.com/kjaebker/Symbiont/pull/63) | Card base extraction: `CardBase` primitives covering glass surface + status strip + header + body slot. 9 card variants refactored. |
| A3 | [#64](https://github.com/kjaebker/Symbiont/pull/64) | Frontend code-splitting: `React.lazy()` + `Suspense` on off-critical-path routes. Bundle size reduced significantly. |
| B1 | [#65](https://github.com/kjaebker/Symbiont/pull/65) | Declarative scope rules: replaced `isAdminRoute` URL switch with `s.admin()` and `s.control()` per-handler decorator wrappers. |
| A4 | [#66](https://github.com/kjaebker/Symbiont/pull/66) | `outlet_program_history` documented as intentional append-only audit log. Not dropped. |
| fix | [#67](https://github.com/kjaebker/Symbiont/pull/67) | `events.initiated_by` propagation fixed: X-Source header now correctly flows through outlet control path. |

---

## What's left, ordered by ROI

### Tier A — ✅ All shipped

#### ~~A1. Schema-drift test~~ — shipped in #62
`internal/enums/drift_test.go` — asserts no drift between `enums.Set` values and SQLite CHECK constraints.

#### ~~A2. Card component base extraction~~ — shipped in #63
`CardBase` primitives cover glass surface + status strip + header + body. All 9 card variants refactored.

#### ~~A3. Frontend bundle code-splitting~~ — shipped in #64
`React.lazy()` + `Suspense` on Settings, LivestockDetail, Journal, History.

#### ~~A4. `outlet_program_history` fate decided~~ — shipped in #66
Documented as intentional append-only audit log. Not dropped — kept for future query use.

---

### Tier B

#### ~~B1. Declarative scope rules~~ — shipped in #65
`s.admin(handler)` and `s.control(handler)` decorator wrappers on individual route registrations. `isAdminRoute` URL switch removed. `write` scope added as a fourth level between `read` and `control`.

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
