# Symbiont — Phase 4: Frontend MVP
> React + TypeScript + Vite, dark mode default, Tremor + uPlot

**Deliverable:** Full local dashboard accessible in browser. Dashboard, History, Outlets, Alerts, and Settings pages all functional with real data. Dark mode default. Mobile-responsive.

---

## 4.1 Project Scaffold

↳ depends on: Phase 2 API fully functional

- [ ] [code] Scaffold Vite + React + TypeScript project in `frontend/`:
  ```bash
  cd frontend && npm create vite@latest . -- --template react-ts
  ```
- [ ] [code] Install dependencies:
  ```bash
  npm install @tremor/react @tanstack/react-query lucide-react uplot
  npm install -D tailwindcss postcss autoprefixer @types/uplot
  npx tailwindcss init -p
  ```
- [ ] [code] Install shadcn/ui:
  ```bash
  npx shadcn-ui@latest init
  ```
  - [ ] Choose: TypeScript, Tailwind, CSS variables, dark mode class strategy
- [ ] [code] Install React Router: `npm install react-router-dom`
- [ ] [config] Configure `tailwind.config.ts`:
  - [ ] `darkMode: 'class'`
  - [ ] Content paths include `./src/**/*.{ts,tsx}` and `./node_modules/@tremor/react/**/*.{js,ts,jsx,tsx}`
- [ ] [code] Set dark class on `<html>` element in `index.html` by default: `<html class="dark">`
- [ ] [config] Configure `vite.config.ts`:
  - [ ] Dev server proxy: `/api` → `http://localhost:8420` (avoids CORS in dev)
  - [ ] Build output to `dist/`
- [ ] [code] Configure `tsconfig.json`:
  - [ ] Strict mode enabled
  - [ ] Path alias: `@/*` → `./src/*`
- [ ] [verify] `npm run dev` starts without errors
- [ ] [verify] `npm run build` produces `dist/` folder
- [ ] [verify] Hot reload works with changes to `App.tsx`

---

## 4.2 API Client and Types

- [ ] [code] Create `src/api/types.ts`:
  - [ ] `Probe` interface: `name`, `display_name`, `type`, `value`, `ts`, `status` (no `unit` from Apex — derive from type if needed at API layer)
  - [ ] `ProbeHistoryPoint` interface: `ts`, `value`
  - [ ] `ProbeHistory` interface: `probe`, `from`, `to`, `interval`, `data`
  - [ ] `Outlet` interface: `id`, `did`, `name`, `display_name`, `type`, `state` (from status[0]), `health` (from status[2]), `watts`, `amps` (correlated from input entries)
  - [ ] `OutletEvent` interface: `id`, `ts`, `outlet_id`, `outlet_name`, `from_state`, `to_state`, `initiated_by`
  - [ ] `AlertRule` interface: all fields from SQLite schema
  - [ ] `SystemStatus` interface: controller, poller, db sub-objects
  - [ ] `ProbeStatus` type: `'normal' | 'warning' | 'critical' | 'unknown'`
  - [ ] `OutletState` type: `'ON' | 'OFF' | 'AON' | 'AOF' | 'TBL' | 'PF1' | 'PF2' | 'PF3' | 'PF4'` (user-facing toggle uses ON/OFF/AUTO, but display shows actual Apex state)
- [ ] [code] Create `src/api/client.ts`:
  - [ ] `BASE_URL` constant: `''` (same origin — Vite proxy handles `/api` in dev)
  - [ ] `getToken()` — reads from `localStorage` with key `symbiont_token`
  - [ ] `setToken(token: string)` — writes to `localStorage`
  - [ ] `apiFetch<T>(path: string, init?: RequestInit): Promise<T>`:
    - [ ] Adds `Authorization: Bearer <token>` header
    - [ ] On 401: clear token, redirect to `/login`
    - [ ] On non-2xx: throws `APIError` with message and code
  - [ ] Typed fetch functions for each endpoint:
    - [ ] `getProbes(): Promise<{ probes: Probe[] }>`
    - [ ] `getProbeHistory(name, params): Promise<ProbeHistory>`
    - [ ] `getOutlets(): Promise<{ outlets: Outlet[] }>`
    - [ ] `setOutletState(id, state): Promise<Outlet>`
    - [ ] `getOutletEvents(outletId?, limit?): Promise<{ events: OutletEvent[] }>`
    - [ ] `getSystemStatus(): Promise<SystemStatus>`
    - [ ] `getAlerts(): Promise<{ rules: AlertRule[] }>`
    - [ ] `createAlert(rule): Promise<AlertRule>`
    - [ ] `updateAlert(id, rule): Promise<AlertRule>`
    - [ ] `deleteAlert(id): Promise<void>`
    - [ ] `getProbeConfig(): Promise<...>`
    - [ ] `updateProbeConfig(name, config): Promise<...>`
    - [ ] `getOutletConfig(): Promise<...>`
    - [ ] `updateOutletConfig(id, config): Promise<...>`
    - [ ] `createToken(label): Promise<{ token: string }>`
    - [ ] `listTokens(): Promise<...>`
    - [ ] `revokeToken(id): Promise<void>`
    - [ ] `triggerBackup(): Promise<...>`

---

## 4.3 App Shell, Routing, and Auth

- [ ] [code] Create `src/main.tsx`:
  - [ ] Wrap app in `QueryClientProvider` (TanStack Query)
  - [ ] Wrap in `BrowserRouter` (React Router)
- [ ] [code] Create `src/App.tsx`:
  - [ ] Route definitions: `/`, `/history`, `/outlets`, `/alerts`, `/settings`, `/login`
  - [ ] Protected route wrapper: redirects to `/login` if no token in localStorage
- [ ] [code] Create `src/pages/Login.tsx`:
  - [ ] Simple token input form (no username/password)
  - [ ] On submit: call `GET /api/system` with provided token to verify
  - [ ] On success: save token, redirect to `/`
  - [ ] On failure: show error message
- [ ] [code] Create `src/components/Layout.tsx`:
  - [ ] Sidebar navigation: Dashboard, History, Outlets, Alerts, Settings
  - [ ] TopBar: system status badge (green dot = `poll_ok`, red = stale), last poll time
  - [ ] Main content area
  - [ ] Responsive: sidebar collapses to bottom nav on mobile
- [ ] [code] Add shadcn/ui components needed for shell:
  - [ ] `npx shadcn-ui@latest add button badge separator`
- [ ] [verify] `/login` page renders and accepts token
- [ ] [verify] After login, Layout renders with sidebar
- [ ] [verify] Sidebar navigation switches between page routes

---

## 4.4 TanStack Query and SSE Setup

- [ ] [code] Create `src/hooks/useSSE.ts`:
  - [ ] Creates `EventSource` with token as query param: `/api/stream?token=<token>`
  - [ ] On `probe_update` event: `queryClient.invalidateQueries({ queryKey: ['probes'] })`
  - [ ] On `outlet_update` event: `queryClient.invalidateQueries({ queryKey: ['outlets'] })`
  - [ ] On `alert_fired` event: trigger toast notification
  - [ ] On `heartbeat`: no-op (keeps connection alive)
  - [ ] On error: attempt reconnect after 5 seconds (exponential backoff)
  - [ ] Cleanup: `es.close()` on component unmount
  - [ ] Mount in `Layout.tsx` so SSE runs on all authenticated pages
- [ ] [code] Create `src/hooks/useProbes.ts`:
  - [ ] `useProbes()` — `useQuery` wrapping `getProbes()`, queryKey `['probes']`
  - [ ] `staleTime: 10_000` — matches poll interval
  - [ ] `refetchInterval: false` — SSE drives refresh
- [ ] [code] Create `src/hooks/useOutlets.ts`:
  - [ ] `useOutlets()` — `useQuery` wrapping `getOutlets()`, queryKey `['outlets']`
  - [ ] `useSetOutlet()` — `useMutation` wrapping `setOutletState()`, invalidates `['outlets']` on success
- [ ] [code] Create `src/hooks/useSystem.ts`:
  - [ ] `useSystemStatus()` — `useQuery` with 30s refetch interval (slower than probes)
- [ ] [verify] Probe values update in browser without page refresh when Apex values change (confirm via SSE events in network tab)

---

## 4.5 Dashboard Page

- [ ] [code] Add shadcn/ui and Tremor components:
  - [ ] `npx shadcn-ui@latest add card`
- [ ] [code] Create `src/components/ProbeCard.tsx`:
  - [ ] Props: `probe: Probe`
  - [ ] Tremor `Card` with metric display: display_name, value + unit, status badge
  - [ ] Status badge color: green=normal, amber=warning, red=critical, gray=unknown
  - [ ] Tremor `Sparkline` for mini trend (loads last 2 hours of history on mount)
  - [ ] Last updated time (relative: "3s ago", "2m ago")
  - [ ] Click anywhere on card → navigate to `/history?probe=<name>`
- [ ] [code] Create `src/components/OutletCard.tsx`:
  - [ ] Props: `outlet: Outlet`
  - [ ] Display: name, state badge (from status[0]), watts, amps (correlated from input entries by API layer)
  - [ ] Toggle button: cycles through ON → OFF → AUTO (or show as 3-state control)
  - [ ] Optimistic update via `useMutation`: update UI immediately, roll back on error
  - [ ] Loading state while mutation in flight
  - [ ] Error state: show brief error message if control fails
- [ ] [code] Create `src/pages/Dashboard.tsx`:
  - [ ] `useProbes()` and `useOutlets()` hooks
  - [ ] Probe grid: responsive grid of `ProbeCard` components
  - [ ] Outlet grid: responsive grid of `OutletCard` components
  - [ ] Loading skeleton while data loads
  - [ ] Empty state if no data (poller not running?)
  - [ ] Sort probes by `display_order` from config
- [ ] [verify] Dashboard loads with real probe values
- [ ] [verify] Values update automatically when SSE event arrives
- [ ] [verify] Outlet toggle sends API request and updates UI
- [ ] [verify] Dashboard is usable on mobile (phone-width)

---

## 4.6 History Page

- [ ] [code] Create uPlot wrapper component `src/components/ProbeChart.tsx`:
  - [ ] Props: `series: Series[]`, `height?: number`
  - [ ] `Series` type: `{ name, data: DataPoint[], unit, color }`
  - [ ] Initializes uPlot in `useEffect` on mount
  - [ ] Destroys and re-creates chart when series count changes (avoids stale refs)
  - [ ] Updates data without re-creating when only values change: `chartRef.current.setData(data)`
  - [ ] Dark mode color scheme (background, grid, text from CSS variables)
  - [ ] Tooltip: shows timestamp + value for each series
  - [ ] Responsive: listens to container width via `ResizeObserver`, calls `chart.setSize`
  - [ ] Legend: series names and colored indicators
  - [ ] Loading state: show skeleton while data fetches
- [ ] [code] Create `src/components/TimeRangePicker.tsx`:
  - [ ] Preset buttons: Last 2h, 6h, 24h, 7d, 30d
  - [ ] Custom range: two datetime inputs
  - [ ] Returns `{ from: Date, to: Date }`
- [ ] [code] Create `src/components/ProbeSelector.tsx`:
  - [ ] Multi-select dropdown of available probe names (from `useProbes`)
  - [ ] Selected probes list with color assignment
  - [ ] Max 4 simultaneous series (uPlot performance)
- [ ] [code] Create `src/pages/History.tsx`:
  - [ ] `ProbeSelector` component (default: first probe or query param probe)
  - [ ] `TimeRangePicker` component (default: last 24h)
  - [ ] Interval selector: Auto, 10s, 1m, 5m, 15m, 1h, 1d
  - [ ] `useQuery` for each selected probe's history (separate query per probe)
  - [ ] Pass all series to `ProbeChart`
  - [ ] Summary stats below chart: min, max, avg for each series
  - [ ] URL sync: `?probe=Temp,pH&from=...&to=...&interval=5m` so links are shareable
- [ ] [verify] History loads with single probe
- [ ] [verify] Multiple probes overlay correctly on same chart
- [ ] [verify] Time range picker changes data range
- [ ] [verify] 24h of 10s data (8,640 points) renders without lag

---

## 4.7 Outlets Page

- [ ] [code] Add shadcn/ui components:
  - [ ] `npx shadcn-ui@latest add table select`
- [ ] [code] Create `src/pages/Outlets.tsx`:
  - [ ] Full outlet table with columns: NAME, STATE, TYPE, HEALTH, WATTS, AMPS, CONTROL
  - [ ] State badge with color coding
  - [ ] CONTROL column: three-button group (ON / OFF / AUTO)
  - [ ] Active state button highlighted
  - [ ] Mutation loading/error state per outlet row
  - [ ] Outlet event log section below table:
    - [ ] Shows last 50 events (ts, outlet name, from→to state, initiated_by)
    - [ ] `initiated_by` badge: UI/CLI/MCP/API
    - [ ] Auto-refreshes when SSE `outlet_update` event arrives
  - [ ] "Show all" pagination if >50 events
- [ ] [verify] All outlets visible with correct states
- [ ] [verify] Outlet control buttons work
- [ ] [verify] Event log updates when outlet is toggled

---

## 4.8 Alerts Page

- [ ] [code] Add shadcn/ui components:
  - [ ] `npx shadcn-ui@latest add dialog form input label select switch`
- [ ] [code] Create `src/components/AlertRuleForm.tsx`:
  - [ ] shadcn/ui Dialog form
  - [ ] Fields: probe (select from available probes), condition (above/below/outside_range), threshold low/high, severity, cooldown, enabled
  - [ ] Validation before submit
  - [ ] Used for both create and edit (controlled by presence of `rule` prop)
- [ ] [code] Create `src/pages/Alerts.tsx`:
  - [ ] Alert rules list as table: PROBE, CONDITION, THRESHOLD, SEVERITY, STATUS, ACTIONS
  - [ ] Status column: enabled/disabled toggle (switch component)
  - [ ] ACTIONS: edit (pencil icon), delete (trash icon with confirmation)
  - [ ] "New Rule" button opens `AlertRuleForm` dialog
  - [ ] Empty state: "No alert rules configured. Add one to get notified when parameters go out of range."
  - [ ] Recent alert events section (fires and clears from `alert_events` SQLite table)
- [ ] [verify] Create, edit, delete alert rules
- [ ] [verify] Enable/disable toggle works

---

## 4.9 Settings Page

- [ ] [code] Add shadcn/ui components:
  - [ ] `npx shadcn-ui@latest add tabs input`
- [ ] [code] Create `src/pages/Settings.tsx` with tabbed layout:

  **Tab: Probes**
  - [ ] Table of probe configs: PROBE, DISPLAY NAME, UNIT OVERRIDE, ORDER, MIN NORMAL, MAX NORMAL, MIN WARNING, MAX WARNING
  - [ ] Inline editing (click cell to edit)
  - [ ] Auto-save on blur

  **Tab: Outlets**
  - [ ] Table of outlet configs: OUTLET ID, DISPLAY NAME, ORDER, HIDDEN
  - [ ] Inline editing

  **Tab: Tokens**
  - [ ] Token list table: ID, LABEL, CREATED, LAST USED
  - [ ] "Create Token" button: label input → creates token → shows token value once in a dismissible alert
  - [ ] Revoke button per row with confirmation

  **Tab: Notifications**
  - [ ] ntfy.sh topic URL input
  - [ ] Test notification button
  - [ ] Enabled/disabled toggle

  **Tab: Backup**
  - [ ] Recent backup jobs table: DATE, STATUS, SIZE, PATH
  - [ ] "Run Backup Now" button
  - [ ] Success/failure feedback

- [ ] [verify] All tabs render without errors
- [ ] [verify] Probe config edits persist (reflect in Dashboard ProbeCard display names)
- [ ] [verify] Token creation and revocation work

---

## 4.10 Polish and Responsive Layout

- [ ] [code] Implement mobile navigation:
  - [ ] Below `md` breakpoint: sidebar becomes bottom navigation bar
  - [ ] Icons only (no labels) in bottom nav
  - [ ] Active route highlighted
- [ ] [code] Loading states:
  - [ ] Skeleton components for ProbeCard, OutletCard, table rows
  - [ ] Spinner for mutations in flight
- [ ] [code] Error states:
  - [ ] API error boundary for page-level failures
  - [ ] Inline error messages for mutations
  - [ ] Toast notifications for SSE alert events (use shadcn/ui Sonner or similar)
- [ ] [code] Empty states for all list views
- [ ] [code] `src/components/StatusBar.tsx`:
  - [ ] Shows in TopBar: last poll time relative ("5s ago"), green/red dot for `poll_ok`
  - [ ] Refreshes every 5 seconds via `useSystemStatus`
- [ ] [code] Favicon: simple wave/reef icon (SVG)
- [ ] [code] Page `<title>` tags: "Symbiont — Dashboard", "Symbiont — History", etc.
- [ ] [verify] All pages render without console errors
- [ ] [verify] Pages are usable at 375px width (iPhone SE)
- [ ] [verify] Pages look correct at 768px (tablet) and 1280px (desktop)

---

## 4.11 Build and Serve Integration

- [ ] [code] `npm run build` in `frontend/` outputs to `frontend/dist/`
- [ ] [config] Update `flake.nix` dev environment to include Node.js for frontend builds
- [ ] [config] Add build step to NixOS package derivation for `symbiont-api` (or build separately and copy)
- [ ] [verify] `go build ./cmd/api` + `SYMBIONT_FRONTEND_PATH=./frontend/dist ./api` serves frontend at `http://localhost:8420/`
- [ ] [verify] React Router routes work (direct navigation to `/history` doesn't 404 — SPA fallback to `index.html`)
- [ ] [verify] API requests from served frontend work (no CORS errors in production mode)
- [ ] [config] Update systemd service if frontend path changes

---

## Phase 4 Checklist Summary

- [ ] Vite + React + TypeScript scaffold
- [ ] API client with all typed fetch functions
- [ ] Auth (login page, token in localStorage)
- [ ] App shell with routing and layout
- [ ] SSE integration driving automatic updates
- [ ] Dashboard: ProbeCard grid + OutletCard grid
- [ ] History: uPlot multi-series chart with time range picker
- [ ] Outlets: full table with controls + event log
- [ ] Alerts: rule management
- [ ] Settings: all tabs functional
- [ ] Mobile-responsive at all breakpoints
- [ ] Build pipeline integrated with API server static serving

**Phase 4 is complete when:** The browser dashboard replaces daily Fusion use. Probe values update automatically. Outlet control works. Dark mode. Usable on phone.
