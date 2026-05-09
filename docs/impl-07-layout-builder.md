# Symbiont — Phase 7: Dashboard Layout Builder
> Customizable dashboard with drag-and-drop ordering

**Deliverable:** Users can customize what appears on their dashboard and in what order. The dashboard is no longer a hardcoded list of all probes and outlets — it's a curated, ordered selection of items the user cares about.

> **Implementation note:** The original Phase 7 spec described a React Flow canvas with draggable probe/outlet nodes representing a physical tank schematic. After evaluating that approach, a simpler and more immediately useful system was built instead: a configurable item list that controls the main dashboard's contents and order. This achieves the customization goal without the complexity of a visual canvas editor.

---

## What Was Built

### 7.1 Backend: `dashboard_items` Table

- [x] [code] Added `dashboard_items` table to SQLite schema:
  ```sql
  CREATE TABLE IF NOT EXISTS dashboard_items (
      id           INTEGER  PRIMARY KEY AUTOINCREMENT,
      item_type    TEXT     NOT NULL,
      reference_id TEXT,
      label        TEXT,
      sort_order   INTEGER  NOT NULL DEFAULT 0,
      display_mode TEXT     NOT NULL DEFAULT 'normal'
  );
  ```
- [x] [code] Item types: `probe` | `outlet` | `device` | `separator` | `feed_mode` | `measurement`
- [x] [code] `reference_id` points to probe name, outlet ID, device ID, or measurement parameter ID depending on type
- [x] [code] `display_mode`: `normal` (full card) or `compact` (condensed row)

### 7.2 Backend: Dashboard API Endpoints

- [x] [code] `GET /api/dashboard` — returns ordered list of dashboard items with live data merged in
- [x] [code] `PUT /api/dashboard` — replaces the full item list (used for reorder saves)
- [x] [code] `POST /api/dashboard` — adds a single item to the dashboard
- [x] [code] `DELETE /api/dashboard/{id}` — removes a single item

### 7.3 Frontend: Dashboard Page

- [x] [code] `GET /api/dashboard` drives the main `/` dashboard
- [x] [code] When no items are configured, dashboard shows all probes and outlets (default behavior)
- [x] [code] Renders a mix of item types in sort order:
  - `probe` → ProbeCard
  - `outlet` → OutletCard
  - `device` → DeviceCard
  - `separator` → visual divider with optional label
  - `feed_mode` → FeedCard (feed mode control)
  - `measurement` → MeasurementCard (latest reading for a parameter)

### 7.4 Frontend: Settings > Dashboard Tab

- [x] [code] `src/pages/settings/DashboardTab.tsx`
- [x] [code] Two-panel layout:
  - Left: current dashboard items in order (drag to reorder, button to remove)
  - Right: available items to add (probes, outlets, devices, separators, feed mode, measurements)
- [x] [code] Drag-and-drop reordering via `@dnd-kit/sortable`
- [x] [code] Each item shows its type, name, and display mode selector (normal/compact)
- [x] [code] Save button calls `PUT /api/dashboard` with the new ordered list
- [x] [code] Changes apply immediately on the main dashboard after save

### 7.5 Hooks and Query Keys

- [x] [code] `useDashboard()` — fetches dashboard items
- [x] [code] `useDashboardReplace()` — mutation for full replace (reorder save)
- [x] [code] `useDashboardAdd()` — mutation for adding a single item
- [x] [code] `useDashboardRemove()` — mutation for removing an item
- [x] [code] Query key: `qk.dashboard.items()`

---

## Phase 7 Checklist

- [x] `dashboard_items` SQLite table and API endpoints
- [x] Six item types supported (probe, outlet, device, separator, feed_mode, measurement)
- [x] Main dashboard renders from `dashboard_items` (falls back to all probes/outlets if empty)
- [x] Settings > Dashboard tab with add/remove/reorder UI
- [x] Drag-and-drop reordering with dnd-kit
- [x] Display mode selector (normal vs compact) per item
- [x] Mobile-responsive (list view works on all screen sizes)

**Phase 7 is complete when:** You can go to Settings > Dashboard, add the probes and outlets you care about in the order you want, and the main dashboard reflects your choices immediately.

---

## What Was Deferred

The React Flow canvas (draggable nodes, physical tank schematic, edge connections between sections) was evaluated and deferred. Reasons:

- The immediate need was "show me what I care about in the right order" — the item list delivers this with far less complexity.
- A canvas editor requires React Flow (~200KB), custom node types, position persistence, and touch-event handling — significant ongoing maintenance cost for a feature used primarily at setup time.
- The item list approach works cleanly on mobile; a canvas editor would not.

If a physical schematic view is desired in the future, it remains feasible as a separate `/layout` page using a canvas library. It would not require changes to the existing dashboard system.
