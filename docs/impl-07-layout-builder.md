# Symbiont — Phase 7: Interactive System Layout Designer

> Visual, interactive aquarium system topology with live data overlays

**Deliverable:** A new `/system` page with an interactive canvas where users build a spatial model of their physical aquarium system — containers, devices, probes, power bars, and the plumbing/electrical/sensor connections between them. View mode shows animated water flow, live probe readings, and outlet states. Edit mode provides drag-and-drop layout building. This is an **additional page** alongside the existing dashboard, not a replacement.

---

## Architecture Decisions

### React Flow (@xyflow/react) as the canvas engine
- Provides zoom/pan, node drag, snap-to-grid, handles, edge routing, minimap out of the box
- Custom node components = full styling control for containers, devices, probes, power bars
- Parent-child nodes (`parentId`) = devices placed inside containers
- Custom edge components = PVC-style orthogonal routing with animated flow particles
- ~30KB gzipped, no conflicts with existing stack

### Data model: JSON blob in SQLite
- Single `system_layouts` table with a JSON `layout` column
- Layout stores nodes (with positions, types, parent relationships) and edges (with layer type, source/target handles)
- References existing data via `deviceId`, `probeName`, `outletId` — not duplicating data

### Three connection layers via CSS toggle
- Each edge gets a `layer-water`, `layer-electrical`, or `layer-sensor` CSS class
- Toggle visibility via CSS class on the React Flow wrapper — no state manipulation needed
- All three visible by default, individually togglable

### T-port splits as nodes, not edge tricks
- A T-port is a small node with 1 input handle and 2+ output handles
- Ball valves are also small nodes with open/closed state
- Standard point-to-point edges connect everything — clean branching

### Flow particle animation via SVG `<animateMotion>`
- Browser-native animation on `<circle>` elements following edge paths
- Zero JS overhead, paused in edit mode via `animation-play-state`

### View mode vs Edit mode
- **View mode** (default): polished live dashboard with animations, flow particles, live data overlays, device glow. No grid, no handles, no resize controls. Click devices to open detail panel.
- **Edit mode**: layout manipulation. Grid visible, anchor points visible, sidebar palette open, no animations. Save/cancel workflow with unsaved changes warning.

---

## Data Model

### TypeScript types (also defines Go struct shape)

```typescript
interface SystemLayout {
  version: 1
  viewport: { x: number; y: number; zoom: number }
  settings: { snapToGrid: boolean; gridSize: number }
  nodes: SystemNode[]
  edges: SystemEdge[]
}

interface SystemNode {
  id: string
  type: 'container' | 'device' | 'probe' | 'powerbar' | 'tport' | 'valve'
  position: { x: number; y: number }
  size?: { width: number; height: number }  // containers, powerbar
  parentId?: string                         // device/probe inside container
  data: {
    label: string
    subtype?: string          // 'tank' | 'sump' | 'ato' for containers; device_type for devices
    deviceId?: number         // links to Device.id
    probeName?: string        // links to Probe.name
    outletId?: string         // for powerbar slots or direct outlet reference
    slots?: PowerBarSlot[]    // powerbar only
  }
}

interface PowerBarSlot {
  index: number       // 1-8
  outletId: string    // maps to Outlet.id
}

interface SystemEdge {
  id: string
  source: string
  sourceHandle?: string
  target: string
  targetHandle?: string
  layer: 'water' | 'electrical' | 'sensor'
  data?: {
    flowDirection?: 'forward' | 'reverse'
  }
}
```

---

## 7.1 Backend — Layout Persistence API

**Files to modify:**
- `internal/db/sqlite_schema.go` — add `system_layouts` table
- `internal/db/sqlite_models.go` — add `SystemLayout` struct
- `internal/db/sqlite_queries.go` — add `GetSystemLayout`, `UpsertSystemLayout`
- `internal/api/server.go` — register routes

**New files:**
- `internal/api/system_layout.go` — GET/PUT handlers
- `internal/api/system_layout_test.go` — roundtrip tests

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS system_layouts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL DEFAULT 'default' UNIQUE,
    layout     TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Endpoints:**
- `GET /api/system/layout` — returns `{ "layout": <json> }` or `{ "layout": null }`
- `PUT /api/system/layout` — accepts `{ "layout": <json> }`, upserts default layout

**Query methods:**
- `GetSystemLayout(ctx, name) (*SystemLayout, error)` — returns nil if not found
- `UpsertSystemLayout(ctx, name, layoutJSON) error` — INSERT OR REPLACE with updated_at

### Tasks
- [x] [code] Add `system_layouts` table to SQLite schema
- [x] [code] Add `SystemLayout` Go model struct
- [x] [code] Add `GetSystemLayout` and `UpsertSystemLayout` query methods
- [x] [code] Add `HandleSystemLayoutGet` and `HandleSystemLayoutSave` API handlers
- [x] [code] Register `GET /api/system/layout` and `PUT /api/system/layout` routes
- [x] [test] Test layout GET/PUT roundtrip via API
- [ ] [verify] `curl` GET returns null, PUT saves, GET returns saved data

---

## 7.2 Frontend — Types, API Client, Hook, Route

**Files to modify:**
- `frontend/src/api/types.ts` — add layout types
- `frontend/src/api/client.ts` — add `getSystemLayout()`, `saveSystemLayout()`
- `frontend/src/App.tsx` — add `/system` route
- `frontend/src/components/Layout.tsx` — add "System" nav item (`Network` icon from lucide-react)

**New files:**
- `frontend/src/hooks/useSystemLayout.ts` — query + mutation hooks

### Tasks
- [x] [code] Add `SystemLayout`, `SystemNode`, `SystemEdge`, `PowerBarSlot` types to `types.ts`
- [x] [code] Add `getSystemLayout()` and `saveSystemLayout()` to API client
- [x] [code] Create `useSystemLayout()` query hook and `useSaveSystemLayout()` mutation hook
- [x] [code] Add `/system` route to `App.tsx`
- [x] [code] Add "System" nav item to sidebar in `Layout.tsx`
- [ ] [verify] Navigate to `/system` — route loads without errors

---

## 7.3 Frontend — Canvas Foundation

**New files:**
- `frontend/src/pages/System.tsx` — top-level page
- `frontend/src/components/system/SystemCanvas.tsx` — React Flow wrapper

**Install:** `npm install @xyflow/react`

**System.tsx responsibilities:**
- Fetch layout via `useSystemLayout()`
- Manage mode state: `'view' | 'edit'`
- Manage layer visibility: `{ water: true, electrical: true, sensor: true }`
- Render toolbar: mode toggle, layer toggles, snap-to-grid toggle (edit only), zoom controls
- Render `<SystemCanvas>` with all props
- Render `<ElementPalette>` sidebar when in edit mode

**SystemCanvas.tsx responsibilities:**
- `<ReactFlowProvider>` wrapping `<ReactFlow>`
- Register `nodeTypes` and `edgeTypes` maps
- Configure `snapToGrid`, `<Background>` (visible in edit, hidden in view)
- Control `nodesDraggable`, `nodesConnectable`, `elementsSelectable` based on mode
- Zoom: scroll wheel. Pan: spacebar+drag (React Flow default behavior)
- `<MiniMap>` styled with dark theme colors
- Drop handler for palette drag-and-drop (detects drop inside container → auto-assigns `parentId`)

**React Flow dark mode CSS:** Override React Flow CSS variables in `index.css` to match surface palette.

### Tasks
- [x] [code] Install `@xyflow/react`
- [x] [code] Create `System.tsx` page with mode toggle, layer toggles, toolbar
- [x] [code] Create `SystemCanvas.tsx` with React Flow setup, nodeTypes, edgeTypes registration
- [x] [code] Add React Flow dark mode CSS overrides to `index.css`
- [ ] [verify] Empty React Flow canvas renders at `/system` with zoom/pan working

---

## 7.4 Frontend — Custom Node Components

All in `frontend/src/components/system/nodes/`

### ContainerNode.tsx
- Large rounded rectangle with glassmorphism fill
- Label at top with subtype icon (lucide: `Box` for tank, `Layers` for sump, `Droplets` for ATO)
- View mode: subtle ambient glow border
- Edit mode: `<NodeResizer>` for drag-to-resize, visible anchor handles
- Handles: Top/Bottom/Left/Right for water connections (both source and target)

### DeviceNode.tsx
- Compact node placed inside containers (`parentId`, `extent: 'parent'`)
- Device icon + name + on/off glow based on linked outlet state
- View mode: primary glow when ON, dimmed when OFF
- Click in view mode → opens detail panel
- Handles: input for electrical (from power bar), input for sensor (from probe)

### ProbeNode.tsx
- Small node placed inside containers
- Probe icon + current reading value + unit + status color dot
- Live reading from probe data via `SystemDataContext`
- Handle: output for sensor connection to device

### PowerBarNode.tsx
- Vertical node with 8 numbered slot rows
- Each slot: number, outlet name, state indicator (green/red/amber)
- Each slot has a `Handle` at `Position.Right` with id `slot-{index}`
- View mode: wattage display per slot if available

### TPortNode.tsx
- Small circle/diamond junction node
- 1 input handle (Left), 2 output handles (Right-top, Right-bottom)
- Visual: PVC T-fitting style

### ValveNode.tsx
- Small node with open/closed visual state
- 1 input handle, 1 output handle
- Visual: ball valve icon, green when open, red when closed

### Tasks
- [x] [code] Create `ContainerNode.tsx` with glassmorphism styling, resize handles, water connection handles
- [x] [code] Create `DeviceNode.tsx` with device icon, on/off glow, electrical + sensor handles
- [x] [code] Create `ProbeNode.tsx` with live reading display, sensor output handle
- [x] [code] Create `PowerBarNode.tsx` with 8 numbered slot rows, per-slot handles
- [x] [code] Create `TPortNode.tsx` with 1-in/2-out handles
- [x] [code] Create `ValveNode.tsx` with open/closed visual state
- [ ] [verify] Each node type renders correctly when manually added to the canvas

---

## 7.5 Frontend — Custom Edge Components

All in `frontend/src/components/system/edges/`

### WaterFlowEdge.tsx
- Orthogonal routing via `getSmoothStepPath` with `borderRadius: 0` (sharp PVC corners)
- Stroke: 3px, primary cyan color (`var(--color-primary)`)
- Directional arrow marker at target end
- View mode: 2-3 animated `<circle>` elements with `<animateMotion>` along the path
- CSS class: `layer-water`

### ElectricalEdge.tsx
- Smooth step routing
- Stroke: 2px, amber color (`#f59e0b`)
- Solid when outlet ON, dashed when OFF (driven by outlet state data)
- Wattage label at midpoint if available
- CSS class: `layer-electrical`

### SensorEdge.tsx
- Thin bezier line, secondary green color (`var(--color-secondary)`)
- Current reading label near source (probe) end
- CSS class: `layer-sensor`

### Tasks
- [x] [code] Create `WaterFlowEdge.tsx` with orthogonal routing, directional arrows, animated flow particles
- [x] [code] Create `ElectricalEdge.tsx` with on/off styling, wattage labels
- [x] [code] Create `SensorEdge.tsx` with reading labels
- [ ] [verify] Each edge type renders correctly between test nodes

---

## 7.6 Frontend — Edit Mode Interactions

**New files:**
- `frontend/src/components/system/ElementPalette.tsx` — sidebar palette

### ElementPalette.tsx
- Left sidebar, visible only in edit mode
- Sections: Containers, Devices, Probes, Power Bars, Plumbing (T-ports, valves)
- Containers: generic "Tank", "Sump", "ATO Reservoir" items
- Devices/Probes: populated from `useDevices()` / `useProbes()`, filters out already-placed items
- Items are HTML-draggable (native drag) — React Flow's `onDrop`/`onDragOver` handles placement
- Each palette item sets `dataTransfer` with node type + reference data

### Connection creation (SystemCanvas.tsx)
- `onConnect` callback determines layer from source/target node types:
  - Power bar → device = `electrical`
  - Container ↔ container (or through T-port/valve) = `water`
  - Probe → device = `sensor`
- Creates edge with appropriate custom edge type

### Save/Cancel (System.tsx)
- Entering edit mode: deep clone current nodes/edges as snapshot
- "Save" button: calls `useSaveSystemLayout()` mutation with serialized React Flow state
- "Cancel" button: restore snapshot, exit edit mode
- `beforeunload` warning when unsaved changes exist

### Node/edge deletion
- Delete/Backspace key removes selected elements (React Flow built-in)
- Connected edges auto-removed when node deleted

### Snap-to-grid toggle
- Toggle button in edit toolbar
- Controls React Flow `snapToGrid` prop + `snapGrid={[20, 20]}`

### Tasks
- [x] [code] Create `ElementPalette.tsx` with sectioned drag palette
- [x] [code] Implement drag-from-palette-to-canvas drop handling with container detection
- [x] [code] Implement `onConnect` with automatic layer type inference
- [x] [code] Implement save/cancel workflow with snapshot and `beforeunload` warning
- [x] [code] Wire up Delete key for node/edge removal
- [x] [code] Implement snap-to-grid toggle
- [ ] [verify] Full edit workflow: drag elements, connect them, save, refresh — layout persists

---

## 7.7 Frontend — View Mode: Live Data & Animations

**New files:**
- `frontend/src/components/system/SystemDataProvider.tsx` — context for live data lookups
- `frontend/src/components/system/DetailPanel.tsx` — slide-over on device/probe click

### SystemDataProvider.tsx
- Context provider wrapping the canvas
- Subscribes to `useProbes()`, `useOutlets()`, `useDevices()` hooks
- Builds lookup maps: `probesByName`, `outletsById`, `devicesById`
- Node components read from context — no per-node queries
- SSE events (via existing `useSSE()` hook) invalidate these queries automatically

### DetailPanel.tsx
- Slide-over panel from right side, opened by clicking device/probe in view mode
- Shows: device info, linked probe readings, sparkline chart, outlet controls
- Reuses patterns from existing `DeviceCard.tsx`
- "View Full Details" link navigates to history page
- Closes on Escape or click outside

### View mode specifics
- Flow particles animate along water edges
- Probe nodes show live readings
- Device nodes glow based on outlet state
- Power bar slots colored by outlet state
- No grid, no handles, no resize controls visible
- Background: clean, no dots

### Tasks
- [x] [code] Create `SystemDataProvider.tsx` with live data context and lookup maps
- [x] [code] Wire all node components to read live data from context
- [ ] [code] Create `DetailPanel.tsx` slide-over with device info, sparkline, outlet controls
- [x] [code] Enable flow particle animations in view mode, pause in edit mode
- [ ] [verify] Probe readings update live when SSE events arrive
- [ ] [verify] Clicking a device opens detail panel with correct data

---

## 7.8 Frontend — Layer Toggles & Polish

### Layer toggle UI (System.tsx toolbar)
- Three toggle buttons with colored indicators: Water (cyan), Electrical (amber), Sensors (green)
- Each toggles a CSS class on the React Flow wrapper div

### CSS layer visibility (index.css)
```css
.hide-layer-water .react-flow__edge.layer-water { display: none }
.hide-layer-electrical .react-flow__edge.layer-electrical { display: none }
.hide-layer-sensor .react-flow__edge.layer-sensor { display: none }
```

### Additional polish
- React Flow CSS variable overrides for dark theme
- Animation keyframes for flow particles
- Minimap with dark theme colors
- Transitions on mode switch (300ms ease-in-out per design system)
- Empty state: when no layout exists, show centered prompt to enter edit mode
- Orphan handling: if a device/probe is deleted from the system, its canvas node shows a warning indicator

### Tasks
- [ ] [code] Implement layer toggle buttons in toolbar
- [ ] [code] Add CSS rules for layer visibility toggling
- [ ] [code] Style minimap with dark theme
- [ ] [code] Add mode switch transitions
- [ ] [code] Create empty state for new users (prompt to enter edit mode)
- [ ] [code] Handle orphaned nodes gracefully (deleted devices/probes)
- [ ] [verify] Layer toggles hide/show only their layer without affecting others

---

## 7.9 Testing

### Backend tests (`internal/api/system_layout_test.go`)
- [ ] [test] GET empty → `{ "layout": null }`
- [ ] [test] PUT valid JSON → 200
- [ ] [test] GET after PUT → returns saved JSON exactly
- [ ] [test] PUT overwrite → returns new JSON
- [ ] [test] PUT invalid JSON → 400

### Frontend tests (`frontend/src/components/system/__tests__/`)
- [ ] [test] SystemCanvas renders with mock nodes/edges
- [ ] [test] Mode toggle switches between view/edit behavior
- [ ] [test] Layer toggles hide/show edges
- [ ] [test] Save mutation called with correct serialized data

---

## Sequencing

```
Phase 7.1 (Backend)  ─────────────────────┐
Phase 7.2 (Types/Client/Route) ───────────┤
                                           ▼
Phase 7.3 (Canvas Foundation) ─────────────┐
                                           ▼
Phase 7.4 (Custom Nodes) ─────┬── parallel ──┬── Phase 7.5 (Custom Edges)
                               ▼              ▼
Phase 7.6 (Edit Mode) ────┬── parallel ──┬── Phase 7.7 (View Mode Live Data)
                           ▼              ▼
Phase 7.8 (Layer Toggles & Polish) ───────┐
                                           ▼
Phase 7.9 (Testing)
```

---

## Verification Checklist

- [ ] **Backend roundtrip:** `curl` GET returns null, PUT saves, GET returns saved data
- [ ] **Empty canvas:** Navigate to `/system`, see empty state prompting to enter edit mode
- [ ] **Edit mode:** Toggle to edit, drag containers/devices from palette, place devices inside containers, connect with edges, save
- [ ] **Persistence:** Refresh page — layout preserved exactly
- [ ] **View mode:** Toggle to view, see animated water flow particles, live probe readings updating, device glow states
- [ ] **Layer toggles:** Toggle each layer off/on, verify edges hide/show without affecting other layers
- [ ] **Detail panel:** Click a device in view mode, see detail panel with controls and data
- [ ] **Zoom/pan:** Scroll to zoom, spacebar+drag to pan
- [ ] **Existing pages unaffected:** Dashboard, History, Outlets, Alerts, Settings all work as before
- [ ] **`go test ./...`** passes
- [ ] **`cd frontend && npm test`** passes
- [ ] **`cd frontend && npx tsc --noEmit`** passes
