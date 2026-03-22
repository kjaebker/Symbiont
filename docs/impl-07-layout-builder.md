# Symbiont — Phase 7: Visual Layout Builder
> Live tank schematic with React Flow

**Deliverable:** An interactive canvas representing the physical tank system. Probe and outlet nodes are draggable and assigned to real equipment. Live data overlays the layout. Click any node to dive deeper.

> **Note:** Full scope should be refined based on real usage of Phases 1–6 before starting. This plan is intentionally more flexible than previous phases. Start simple and iterate.

---

## 7.1 Scoping Review (Before Starting)

Before writing any code, answer these questions based on actual usage of the dashboard:

- [ ] [decision] What does the physical layout actually need to show? (Sump, display tank, frag tank, equipment rack?)
- [ ] [decision] What node types are needed? (Probe indicator, outlet tile, equipment label, section boundary?)
- [ ] [decision] Is free-form drag-and-drop positioning needed, or is a simpler grid/section layout sufficient?
- [ ] [decision] Should edges (flow arrows between sections) be supported in v1, or deferred?
- [ ] [decision] Should node shapes be customizable, or is text-labeled rectangles sufficient?
- [ ] [research] Evaluate React Flow v12 — confirm it handles the node types needed and dark mode theming
- [ ] [decision] Confirm SQLite JSON column is the right storage for layout config (vs. normalized tables)

---

## 7.2 Backend: Layout API

↳ depends on: 7.1 scoping complete

- [ ] [code] Add `layout` table to SQLite schema:
  ```sql
  CREATE TABLE IF NOT EXISTS layouts (
      id          INTEGER  PRIMARY KEY AUTOINCREMENT,
      name        TEXT     NOT NULL DEFAULT 'Default',
      config      TEXT     NOT NULL,  -- JSON blob: nodes, edges, viewport
      created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
  );
  ```
- [ ] [code] Add SQLite queries:
  - [ ] `GetDefaultLayout() (*Layout, error)` — returns first layout, or empty config if none
  - [ ] `SaveLayout(config string) error` — upsert default layout
  - [ ] `ListLayouts() ([]LayoutMeta, error)` — id, name, updated_at
- [ ] [code] Add API endpoints:
  - [ ] `GET /api/layout` — returns current layout config JSON
  - [ ] `PUT /api/layout` — saves layout config JSON (full replace)
- [ ] [test] Test layout get/save roundtrip via API
- [ ] [verify] `curl -X PUT /api/layout -d '{"nodes":[],"edges":[]}'` saves and retrieves correctly

---

## 7.3 Frontend: Dependencies and Setup

- [ ] [code] Install React Flow:
  ```bash
  npm install @xyflow/react
  ```
- [ ] [code] Import React Flow CSS in `main.tsx`:
  ```typescript
  import '@xyflow/react/dist/style.css';
  ```
- [ ] [code] Configure React Flow dark mode:
  - [ ] Use CSS variables to override React Flow's default light colors
  - [ ] Background, node, edge, handle colors match Symbiont dark theme
- [ ] [code] Add `/layout` route to `App.tsx`
- [ ] [code] Add "Layout" nav item to sidebar (with a "beta" badge)
- [ ] [verify] Empty React Flow canvas renders at `/layout` without errors

---

## 7.4 Node Types

- [ ] [code] Create `src/components/layout/ProbeNode.tsx`:
  - [ ] Props: React Flow node props + `data: { probe_name: string }`
  - [ ] Renders: probe display name, current value + unit, status color indicator
  - [ ] Status ring: green/amber/red border based on probe status
  - [ ] Click in view mode: opens probe history drawer
  - [ ] Edit mode: shows configuration handle (probe_name selector)

- [ ] [code] Create `src/components/layout/OutletNode.tsx`:
  - [ ] Props: React Flow node props + `data: { outlet_id: string }`
  - [ ] Renders: outlet display name, state badge (ON/OFF/AON/AOF), watts
  - [ ] Click in view mode: shows outlet control popover (ON/OFF toggle)
  - [ ] Edit mode: shows configuration handle (outlet_id selector)

- [ ] [code] Create `src/components/layout/LabelNode.tsx`:
  - [ ] Props: React Flow node props + `data: { text: string, style?: 'section' | 'equipment' }`
  - [ ] Renders: text label only — for naming sections ("Display Tank", "Sump") or equipment ("Skimmer", "Return Pump")
  - [ ] `section` style: larger text, subtle border, acts as visual container
  - [ ] `equipment` style: smaller text, simpler appearance
  - [ ] Edit mode: inline text editing on double-click

- [ ] [code] Register all node types in React Flow:
  ```typescript
  const nodeTypes = {
    probe: ProbeNode,
    outlet: OutletNode,
    label: LabelNode,
  };
  ```

---

## 7.5 Layout Canvas Page

- [ ] [code] Create `src/pages/Layout.tsx`:
  - [ ] Mode toggle: **View** (default) vs **Edit** — prominent button in top-right
  - [ ] `ReactFlow` component with `nodeTypes`, `nodes`, `edges`, `onNodesChange`, `onEdgesChange`
  - [ ] `Background` component: subtle dot grid
  - [ ] `MiniMap` component: collapsed by default, toggle button
  - [ ] `Controls` component: zoom in/out, fit view, lock

  **View mode behavior:**
  - [ ] Nodes are not draggable (`nodesDraggable={false}`)
  - [ ] Edges are not editable (`elementsSelectable={false}`)
  - [ ] Live data overlaid on nodes via `useProbes()` and `useOutlets()` hooks
  - [ ] SSE updates → TanStack Query invalidation → nodes re-render with new values
  - [ ] Click probe node → opens `ProbeHistoryDrawer`
  - [ ] Click outlet node → shows outlet control popover

  **Edit mode behavior:**
  - [ ] Nodes are draggable
  - [ ] Toolbar appears: "Add Probe", "Add Outlet", "Add Label", "Delete Selected"
  - [ ] Selecting a node shows config panel (right sidebar or floating)
  - [ ] Drag nodes freely, positions snap to grid (optional)
  - [ ] "Save Layout" button — calls `PUT /api/layout`
  - [ ] "Cancel" button — reverts to last saved state
  - [ ] Unsaved changes indicator

---

## 7.6 Edit Mode Toolbar and Node Config

- [ ] [code] Create `src/components/layout/LayoutToolbar.tsx`:
  - [ ] Add Probe button: inserts a `probe` node at center of viewport
  - [ ] Add Outlet button: inserts an `outlet` node
  - [ ] Add Label button: inserts a `label` node with "New Label" text
  - [ ] Delete button: deletes selected nodes (Backspace key also works)
  - [ ] Save button: serializes layout and `PUT /api/layout`
  - [ ] Cancel button: reloads from saved state

- [ ] [code] Create `src/components/layout/NodeConfigPanel.tsx`:
  - [ ] Appears when a node is selected in edit mode
  - [ ] Positioned as floating panel (right side of canvas, or bottom)
  - [ ] **For probe nodes:**
    - [ ] Dropdown: select which probe to display (from `useProbes()` data)
    - [ ] Label override field (optional custom label)
  - [ ] **For outlet nodes:**
    - [ ] Dropdown: select which outlet to display (from `useOutlets()` data)
    - [ ] Label override field
  - [ ] **For label nodes:**
    - [ ] Text input (also editable inline via double-click on node)
    - [ ] Style selector: section / equipment

---

## 7.7 Probe History Drawer

- [ ] [code] Create `src/components/layout/ProbeHistoryDrawer.tsx`:
  - [ ] Triggered by clicking a probe node in view mode
  - [ ] Slides in from the right (shadcn/ui Sheet component)
  - [ ] Content:
    - [ ] Probe display name and current value
    - [ ] Time range selector (presets: 2h, 6h, 24h, 7d)
    - [ ] `ProbeChart` (uPlot) with history data for selected range
    - [ ] Min/max/avg stats for the range
    - [ ] Link to full History page for this probe
  - [ ] Closes on Escape or click outside

- [ ] [code] Add shadcn/ui Sheet:
  - [ ] `npx shadcn-ui@latest add sheet`

- [ ] [verify] Clicking a probe node opens drawer with live chart

---

## 7.8 Layout Persistence

- [ ] [code] Layout config stored as JSON in SQLite:
  ```json
  {
    "nodes": [
      {
        "id": "probe-Temp",
        "type": "probe",
        "position": { "x": 100, "y": 200 },
        "data": { "probe_name": "Temp" }
      }
    ],
    "edges": [],
    "viewport": { "x": 0, "y": 0, "zoom": 1 }
  }
  ```
- [ ] [code] On `GET /api/layout`: return stored JSON or `{ "nodes": [], "edges": [], "viewport": {} }` if no layout saved
- [ ] [code] On page load: fetch layout config, initialize React Flow with stored nodes/edges/viewport
- [ ] [code] On save: serialize React Flow state to JSON, `PUT /api/layout`
- [ ] [verify] Save layout, refresh page — positions and node assignments are preserved
- [ ] [verify] Save layout, view on a different browser — same layout appears

---

## 7.9 Live Data Integration

- [ ] [code] In `Layout.tsx`: merge live probe data into node `data` fields:
  ```typescript
  const { data: probes } = useProbes();
  const { data: outlets } = useOutlets();

  const liveNodes = useMemo(() => nodes.map(node => {
    if (node.type === 'probe') {
      const probe = probes?.probes.find(p => p.name === node.data.probe_name);
      return { ...node, data: { ...node.data, probe } };
    }
    if (node.type === 'outlet') {
      const outlet = outlets?.outlets.find(o => o.id === node.data.outlet_id);
      return { ...node, data: { ...node.data, outlet } };
    }
    return node;
  }), [nodes, probes, outlets]);
  ```
- [ ] [code] `ProbeNode` renders from `data.probe` (live Probe object, can be undefined while loading)
- [ ] [code] `OutletNode` renders from `data.outlet` (live Outlet object)
- [ ] [verify] Probe node values update every 10 seconds when SSE event arrives
- [ ] [verify] Changing an outlet state via the popover updates the node badge immediately (optimistic update)

---

## 7.10 Mobile Considerations for Layout View

- [ ] [decision] On mobile: is the layout canvas useful (touch-to-pan/zoom) or should it be replaced with the standard dashboard?
- [ ] [code] If layout enabled on mobile:
  - [ ] Touch events work in React Flow for pan and zoom
  - [ ] Tap on node opens drawer/modal (no hover states on touch)
  - [ ] Edit mode hidden on mobile (too complex for touch)
- [ ] [code] If layout hidden on mobile:
  - [ ] Hide "Layout" nav item below `md` breakpoint
  - [ ] Or show read-only static version (no pan/zoom, simplified nodes)

---

## 7.11 Future Scope (Not Phase 7)

Explicitly deferred — do not implement in Phase 7:

- Multi-layout support (switching between saved layouts)
- Custom node shapes beyond the three types
- Animated flow arrows between sections
- Photo background (tank photo as canvas background)
- Sharing layouts between users
- Import/export layout as JSON file

---

## Phase 7 Checklist Summary

- [ ] Scoping review complete — decisions made before coding
- [ ] SQLite layout table and API endpoints
- [ ] React Flow integrated with dark mode
- [ ] Three node types: probe, outlet, label
- [ ] View mode: live data, click-through to history drawer
- [ ] Edit mode: toolbar, drag-and-drop, node config panel, save
- [ ] Layout persistence across page refreshes
- [ ] Probe history drawer with uPlot chart
- [ ] Mobile decision made and implemented

**Phase 7 is complete when:** You can build a schematic of your actual tank system in edit mode, save it, and use it as a live-updating control panel in view mode — tapping a probe to see its history, toggling outlets directly from the canvas.
