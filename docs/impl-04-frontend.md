# Symbiont — Phase 4: Frontend MVP
> React + TypeScript + Vite, dark mode default, Tremor + uPlot

**Deliverable:** Full local dashboard accessible in browser. Dashboard, History, Outlets, Alerts, and Settings pages all functional with real data. Dark mode default. Mobile-responsive.

---

## 4.1 Project Scaffold

↳ depends on: Phase 2 API fully functional

- [x] [code] Scaffold Vite + React + TypeScript project in `frontend/`:
  ```bash
  cd frontend && npm create vite@latest . -- --template react-ts
  ```
- [x] [code] Install dependencies:
  ```bash
  npm install @tanstack/react-query lucide-react uplot react-router-dom clsx tailwind-merge
  npm install -D tailwindcss @tailwindcss/vite
  ```
  - Note: Tailwind v4 used (CSS-first config via `@theme`), no Tremor (custom components instead), no shadcn/ui (hand-built with design system)
- [x] [code] Set dark class on `<html>` element in `index.html` by default: `<html class="dark">`
- [x] [config] Configure `vite.config.ts`:
  - [x] Dev server proxy: `/api` → `http://localhost:8420` (avoids CORS in dev)
  - [x] Build output to `dist/`
  - [x] Tailwind v4 plugin via `@tailwindcss/vite`
- [x] [code] Configure `tsconfig.app.json`:
  - [x] Strict mode enabled
  - [x] Path alias: `@/*` → `./src/*`
- [x] [code] Custom Abyssal Laboratory design system in `src/index.css`:
  - [x] Full surface hierarchy, primary/secondary/tertiary palette via `@theme`
  - [x] Manrope font, custom utilities (glass, animate-bio-pulse, shadow-glow-*, transition-fluid)
- [x] [verify] `npm run build` produces `dist/` folder (315KB JS + 22KB CSS)

---

## 4.2 API Client and Types

- [x] [code] Create `src/api/types.ts`:
  - [x] `Probe` interface: `name`, `display_name`, `type`, `value`, `unit`, `ts`, `status`
  - [x] `ProbeHistoryPoint` interface: `ts`, `value`
  - [x] `ProbeHistory` interface: `probe`, `from`, `to`, `interval`, `data`
  - [x] `Outlet` interface: `id`, `name`, `display_name`, `type`, `state`, `intensity`
    - Note: No `watts`/`amps` — API doesn't correlate power inputs to individual outlets
  - [x] `OutletEvent` interface: `id`, `ts`, `outlet_id`, `outlet_name`, `from_state`, `to_state`, `initiated_by`
  - [x] `AlertRule` interface: all fields from SQLite schema
  - [x] `SystemStatus` interface: controller, poller, db sub-objects
  - [x] `ProbeStatus` type: `'normal' | 'warning' | 'critical' | 'unknown'`
  - [x] `OutletState` type: `'ON' | 'OFF' | 'AON' | 'AOF' | 'TBL' | 'PF1' | 'PF2' | 'PF3' | 'PF4'`
  - [x] `ProbeConfig`, `OutletConfig`, `BackupJob`, `AuthToken`, `APIError` interfaces
- [x] [code] Create `src/api/client.ts`:
  - [x] `getToken()` / `setToken()` / `clearToken()` — localStorage with key `symbiont_token`
  - [x] `apiFetch<T>()` — Bearer auth, 401 redirect, error throwing via `APIRequestError`
  - [x] All typed fetch functions: `getProbes`, `getProbeHistory`, `getOutlets`, `setOutletState`, `getOutletEvents`, `getSystemStatus`, `getAlerts`, `createAlert`, `updateAlert`, `deleteAlert`, `getProbeConfigs`, `updateProbeConfig`, `getOutletConfigs`, `updateOutletConfig`, `listTokens`, `createToken`, `revokeToken`, `getBackups`, `triggerBackup`

---

## 4.3 App Shell, Routing, and Auth

- [x] [code] Create `src/main.tsx`:
  - [x] Wrap app in `QueryClientProvider` (TanStack Query)
  - [x] Wrap in `BrowserRouter` (React Router)
- [x] [code] Create `src/App.tsx`:
  - [x] Route definitions: `/`, `/history`, `/outlets`, `/alerts`, `/settings`, `/login`
  - [x] Protected route wrapper: `RequireAuth` redirects to `/login` if no token
- [x] [code] Create `src/pages/Login.tsx`:
  - [x] Token input form with Abyssal Laboratory aesthetic
  - [x] On submit: call `GET /api/system` with provided token to verify
  - [x] On success: save token, redirect to `/`
  - [x] On failure: show error message
- [x] [code] Create `src/components/Layout.tsx`:
  - [x] Sidebar navigation: Dashboard, History, Outlets, Alerts, Settings (lucide-react icons)
  - [x] StatusBadge: green/red dot for `poll_ok`, relative time for last poll
  - [x] Main content area via React Router `<Outlet />`
  - [x] Responsive: sidebar on desktop, bottom nav on mobile
  - [x] Active route highlighting with glow effect

---

## 4.4 TanStack Query and SSE Setup

- [x] [code] Create `src/hooks/useSSE.ts`:
  - [x] Creates `EventSource` with token as query param: `/api/stream?token=<token>`
  - [x] On `probe_update` event: `queryClient.invalidateQueries({ queryKey: ['probes'] })`
  - [x] On `outlet_update` event: `queryClient.invalidateQueries({ queryKey: ['outlets'] })`
  - [x] On `alert_fired` event: `queryClient.invalidateQueries({ queryKey: ['alerts'] })`
  - [x] On error: exponential backoff reconnect (1s → 30s max)
  - [x] Cleanup: `es.close()` on component unmount
  - [x] Mounted in `Layout.tsx` so SSE runs on all authenticated pages
- [x] [code] Create `src/hooks/useProbes.ts`:
  - [x] `useProbes()` — `useQuery` wrapping `getProbes()`, queryKey `['probes']`, staleTime 10s
  - [x] `useProbeHistory()` — `useQuery` wrapping `getProbeHistory()`, staleTime 30s
- [x] [code] Create `src/hooks/useOutlets.ts`:
  - [x] `useOutlets()` — `useQuery` wrapping `getOutlets()`, queryKey `['outlets']`
  - [x] `useSetOutlet()` — `useMutation` with optimistic updates and rollback on error
  - [x] `useOutletEvents()` — `useQuery` wrapping `getOutletEvents()`
- [x] [code] Create `src/hooks/useSystem.ts`:
  - [x] `useSystemStatus()` — `useQuery` with 30s refetch interval

---

## 4.5 Dashboard Page

- [x] [code] Create `src/components/ProbeCard.tsx`:
  - [x] Props: `probe: Probe`
  - [x] Custom card with display_name, value + unit, status indicator (bioluminescent pulse)
  - [x] Status indicator color: green=normal, amber=warning, red=critical, gray=unknown
  - [x] Custom SVG `Sparkline` component (loads last 2 hours of history via `useQuery`)
  - [x] Click anywhere on card → navigate to `/history?probe=<name>`
- [x] [code] Create `src/components/Sparkline.tsx`:
  - [x] SVG sparkline with gradient fill and glow effect
  - [x] Placeholder bars when no data
- [x] [code] Create `src/components/OutletCard.tsx`:
  - [x] Props: `outlet: Outlet`
  - [x] Display: name, state label, icon (Zap when on, Power when off)
  - [x] Two-button control group: OFF / ON with active state highlighting
  - [x] Optimistic update via `useSetOutlet()` mutation with rollback
  - [x] Disabled state while mutation in flight
- [x] [code] Create `src/pages/Dashboard.tsx`:
  - [x] Welcome header with summary stats (active modules, critical alerts count)
  - [x] Overall status badge (Stable/Warning/Critical — worst of all probes)
  - [x] Probe grid: responsive 1/2/4 column grid of ProbeCards
  - [x] Power Management: responsive 1/2 column grid of OutletCards (filtered to type=outlet)
  - [x] System Events: recent outlet events timeline with relative times and initiated_by badges
  - [x] System Health card: poll status, controller serial, firmware
  - [x] Loading skeleton while data loads
  - [x] Empty states for outlets and events

---

## 4.6 History Page

- [x] [code] Create uPlot wrapper component `src/components/ProbeChart.tsx`:
  - [x] Props: `series: Series[]`, `height?: number`
  - [x] `Series` type: `{ name, data: DataPoint[], unit, color }`
  - [x] Initializes uPlot in `useEffect` on mount
  - [x] Destroys and re-creates chart when series count changes (avoids stale refs)
  - [x] Updates data without re-creating when only values change: `chartRef.current.setData(data)`
  - [x] Dark mode color scheme (background, grid, text from CSS variables)
  - [x] Tooltip: shows timestamp + value for each series
  - [x] Responsive: listens to container width via `ResizeObserver`, calls `chart.setSize`
  - [x] Legend: series names and colored indicators
  - [x] Loading state: show skeleton while data fetches
- [x] [code] Create `src/components/TimeRangePicker.tsx`:
  - [x] Preset buttons: Last 2h, 6h, 24h, 7d, 30d
  - [x] Custom range: two datetime inputs
  - [x] Returns `{ from: Date, to: Date }`
- [x] [code] Create `src/components/ProbeSelector.tsx`:
  - [x] Multi-select dropdown of available probe names (from `useProbes`)
  - [x] Selected probes list with color assignment
  - [x] Max 4 simultaneous series (uPlot performance)
- [x] [code] Create `src/pages/History.tsx`:
  - [x] `ProbeSelector` component (default: first probe or query param probe)
  - [x] `TimeRangePicker` component (default: last 24h)
  - [x] Interval selector: Auto, 10s, 1m, 5m, 15m, 1h, 1d
  - [x] `useQuery` for each selected probe's history (separate query per probe)
  - [x] Pass all series to `ProbeChart`
  - [x] Summary stats below chart: min, max, avg for each series
  - [x] URL sync: `?probe=Temp,pH&from=...&to=...&interval=5m` so links are shareable
- [x] [verify] History loads with single probe
- [x] [verify] Multiple probes overlay correctly on same chart
- [x] [verify] Time range picker changes data range
- [x] [verify] 24h of 10s data (8,640 points) renders without lag

---

## 4.7 Outlets Page

- [x] [code] Add shadcn/ui components:
  - [x] Skipped — hand-built table with Tailwind matching design system (no shadcn/ui needed)
- [x] [code] Create `src/pages/Outlets.tsx`:
  - [x] Full outlet table with columns: NAME, STATE, TYPE, CONTROL (HEALTH/WATTS/AMPS omitted — not in API)
  - [x] State badge with color coding
  - [x] CONTROL column: three-button group (ON / OFF / AUTO)
  - [x] Active state button highlighted
  - [x] Mutation loading/error state per outlet row
  - [x] Outlet event log section below table:
    - [x] Shows last 50 events (ts, outlet name, from→to state, initiated_by)
    - [x] `initiated_by` badge: UI/CLI/MCP/API
    - [x] Auto-refreshes when SSE `outlet_update` event arrives
  - [x] "Show more" pagination if >50 events
- [x] [verify] All outlets visible with correct states
- [x] [verify] Outlet control buttons work
- [x] [verify] Event log updates when outlet is toggled

---

## 4.8 Alerts Page

- [x] [code] Add shadcn/ui components:
  - [x] Skipped — hand-built with Tailwind matching design system (no shadcn/ui needed)
- [x] [code] Create `src/components/AlertRuleForm.tsx`:
  - [x] Modal dialog form (hand-built, no shadcn/ui)
  - [x] Fields: probe (select from available probes), condition (above/below/outside_range), threshold low/high, severity, cooldown, enabled
  - [x] Validation before submit
  - [x] Used for both create and edit (controlled by presence of `rule` prop)
- [x] [code] Create `src/pages/Alerts.tsx`:
  - [x] Alert rules list as table: PROBE, CONDITION, THRESHOLD, SEVERITY, STATUS, ACTIONS
  - [x] Status column: enabled/disabled toggle (switch component)
  - [x] ACTIONS: edit (pencil icon), delete (trash icon with confirmation)
  - [x] "New Rule" button opens `AlertRuleForm` dialog
  - [x] Empty state: "No alert rules configured. Add one to get notified when parameters go out of range."
  - [ ] Recent alert events section (fires and clears from `alert_events` SQLite table) — pending backend API endpoint
- [x] [verify] Create, edit, delete alert rules
- [x] [verify] Enable/disable toggle works

---

## 4.9 Settings Page

- [x] [code] Add shadcn/ui components:
  - [x] Skipped — hand-built tabs with Tailwind matching design system (no shadcn/ui needed)
- [x] [code] Create `src/pages/Settings.tsx` with tabbed layout:

  **Tab: Probes**
  - [x] Table of probe configs: PROBE, DISPLAY NAME, UNIT OVERRIDE, ORDER, MIN NORMAL, MAX NORMAL, MIN WARNING, MAX WARNING
  - [x] Inline editing (click cell to edit)
  - [x] Auto-save on blur

  **Tab: Outlets**
  - [x] Table of outlet configs: OUTLET ID, DISPLAY NAME, ORDER, HIDDEN
  - [x] Inline editing

  **Tab: Tokens**
  - [x] Token list table: ID, LABEL, CREATED, LAST USED
  - [x] "Create Token" button: label input → creates token → shows token value once in a dismissible alert
  - [x] Revoke button per row with confirmation

  **Tab: Notifications**
  - [x] ntfy.sh topic URL input
  - [x] Test notification button
  - [x] Enabled/disabled toggle

  **Tab: Backup**
  - [x] Recent backup jobs table: DATE, STATUS, SIZE, PATH
  - [x] "Run Backup Now" button
  - [x] Success/failure feedback

- [x] [verify] All tabs render without errors
- [x] [verify] Probe config edits persist (reflect in Dashboard ProbeCard display names)
- [x] [verify] Token creation and revocation work

---

## 4.10 Polish and Responsive Layout

- [x] [code] Implement mobile navigation:
  - [x] Below `md` breakpoint: sidebar becomes bottom navigation bar
  - [x] Icons + labels in bottom nav
  - [x] Active route highlighted with primary color
- [x] [code] Loading states:
  - [x] Skeleton components for ProbeCard, OutletCard
  - [x] Disabled state for mutation buttons in flight
- [x] [code] Error states:
  - [x] API error boundary for page-level failures
  - [x] Inline error messages for mutations
  - [x] Toast notifications for SSE alert events
- [x] [code] Empty states for dashboard list views
- [x] [code] StatusBadge in Layout sidebar:
  - [x] Shows last poll time relative ("5s ago"), green/red dot for `poll_ok`
  - [x] Refreshes via `useSystemStatus` with 30s refetch interval
- [x] [code] Favicon: wave SVG icon
- [x] [code] Page `<title>` tags: "Symbiont — Dashboard", "Symbiont — History", etc.
- [x] [code] Dashboard customization:
  - [x] Allow user to show/hide individual probes from the dashboard grid
  - [x] Persist visibility preferences (SQLite probe config)
  - [x] UI control (Settings → Dashboard tab with drag-and-drop ordering and visibility toggles)
- [x] [verify] All pages render without console errors
- [x] [verify] Pages are usable at 375px width (iPhone SE)
- [x] [verify] Pages look correct at 768px (tablet) and 1280px (desktop)

---

## 4.11 Build and Serve Integration

- [x] [code] `npm run build` in `frontend/` outputs to `frontend/dist/`
- [x] [config] `flake.nix` already includes `nodejs_22` for frontend builds
- [x] [config] Add build step to NixOS package derivation for `symbiont-api` (or build separately and copy)
- [x] [verify] `go build ./cmd/api` + `SYMBIONT_FRONTEND_PATH=./frontend/dist ./api` serves frontend at `http://localhost:8420/`
- [x] [verify] React Router routes work (direct navigation to `/history` doesn't 404 — SPA fallback to `index.html`)
- [x] [verify] API requests from served frontend work (no CORS errors in production mode)
- [x] [config] Update systemd service if frontend path changes

---

## Phase 4 Checklist Summary

- [x] Vite + React + TypeScript scaffold
- [x] API client with all typed fetch functions
- [x] Auth (login page, token in localStorage)
- [x] App shell with routing and layout
- [x] SSE integration driving automatic updates
- [x] Dashboard: ProbeCard grid + OutletCard grid
- [x] History: uPlot multi-series chart with time range picker
- [x] Outlets: full table with controls + event log
- [x] Alerts: rule management
- [x] Settings: all tabs functional (except Notifications — pending backend API)
- [x] Mobile-responsive at all breakpoints
- [x] Build pipeline integrated with API server static serving

**Phase 4 is complete when:** The browser dashboard replaces daily Fusion use. Probe values update automatically. Outlet control works. Dark mode. Usable on phone.
