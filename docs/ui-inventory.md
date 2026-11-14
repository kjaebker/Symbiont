# Symbiont UI Inventory

Snapshot of the frontend as of 2026-11-03 (main @ `bc2af73`), taken from the live app
(desktop 1440 px and phone 390 px) and from the source. This is the baseline for the
UI rebuild: every screen, state, component and behavior that exists today, what works,
what is broken, and what must survive the redesign.

Legend: **KEEP** = signature/working, carry forward · **FIX** = keep the feature, change
the design · **BUG** = behaves wrongly today · **CUT?** = candidate to remove or hide.

---

## 1. Global shell

| Element | Today | Notes |
|---|---|---|
| Desktop sidebar | 7 items: Dashboard, Livestock, Chemistry, Maintenance, Log, Alerts, Settings. Logo, poll status ("9s ago") at bottom | Active item tinted with per-page accent |
| Sidebar silhouette | Animated SVG art at the bottom of the sidebar, different per page (plants, fish, test tubes, kelp/jellyfish, bare branches) | **KEEP** — character. Uses hex colors inline |
| Per-page accent | Dashboard cyan, Livestock green, Chemistry violet `#a78bfa`, Maintenance amber, Log blue `#60a5fa`, Alerts coral, Settings grey (`lib/pageTheme.ts`) | Violet/blue are off-palette; page subtitle is always cyan regardless |
| Caustic background | Two or three large blurred color blobs drifting slowly; Dashboard re-tints them by tank status (cyan / amber / coral) | **KEEP** (status-driven ambient light is a good idea) |
| Bubbles | Floating bubble animation, toggle in Settings › System › Interface | **KEEP** as option |
| Phone nav | Bottom bar: hamburger, current page name, 7 unlabeled dots; swipe the bar to step pages; hamburger opens a push-aside drawer (content slides right) | **FIX** — dots are unreadable; ~30 px dead space under the bar |
| Page header (`PageHeader`) | Cyan eyebrow + 36 px title, optional primary action (becomes a floating pill on phone), underline tabs (desktop) / segmented control (≤4 tabs) / select (5+ tabs) on phone, filter strip (desktop) / Filters bottom sheet (phone), status pills | **FIX** — good shared component, but Livestock detail, Control and Journal bypass it; tab underline is a 1 px border |
| Lazy route fallback | Empty `<div>` while a route chunk loads → blank page on first visit to each page | **FIX** (skeleton) |
| Login | "Subaquatic controller access point", token field, "Establish Connection" | Fine; copy is theatrical |

## 2. Dashboard (`/`)

**Layout is user-defined** in Settings › Dashboard: ordered list of items, each a
probe, outlet, device, feed mode or measurement, each `normal` or `compact`, with
named **separators** that become section headers ("Modes", "Chemistry"). This is the
Phase 7 layout builder. The redesign must render *any* layout, not a fixed page.

Fixed parts above the user layout:

| Part | Behavior | Status |
|---|---|---|
| Tank status header | Headline from worst probe status: "All clear." / "Something's off." / "Check your tank."; detail line (outlets running, critical count, sick and quarantine counts, "Feed active"), amber "N tasks due" link, last-poll time, Lock/Unlocked toggle | **FIX** — never names the problem; ignores alert events and poller state; counts only tasks, not doses |
| Floating lock pill (phone) | Second lock toggle fixed above bottom bar | **BUG/FIX** — duplicate; covers card content |
| Daily check-in | Glass card with a question of the day; types: thumbs (Yes/No), 1–5 rating, free text; Skip; answer goes to Journal | **KEEP**, but it sits above all data and takes the whole first phone screen |

Card types (all in `components/`):

| Card | Normal mode | Compact mode | Tap | Status |
|---|---|---|---|---|
| **DeviceCard** | Blob-shaped icon tinted by device personality, name, type label, mode (AUTO / AUTO OFF / ON), primary reading as hero, secondary readings (Amps, Watts) stacked right, **sparkline** of primary probe (24 h) in personality color; lights show **LightCycleChart** (6 h of each intensity channel, colored lines, legend "COLOR / INTENSITY"); binary probes show text ("Water Level Ok"); OFF/ON/AUTO row slides open when unlocked | `DeviceCompactCard`: icon, name, primary value | Value → Telemetry | **KEEP** — the signature card. **BUG**: tap lands on empty chart (see §9) |
| **OutletCard** | Outlet name, state, OFF/ON/AUTO when unlocked | `OutletCompactCard` | — | Modes row on Dashboard: "Maint With Ret" vs "Maintenance With R…" are near-duplicates |
| **ProbeCard** | Category-tinted card (temp/pH/power/…), value, sparkline | `ProbeCompactCard` | Value → Telemetry | Not on current layout |
| **FeedCard** | Feed A–D buttons, active state with countdown, cancel | `FeedCompactCard` ("INACTIVE") | — | Label "INACTIVE" vs "OFF" elsewhere |
| **MeasurementCard** | Latest manual test value + unit, parameter name, green-tinted | `MeasurementCompactCard` | → `/measurements` (legacy redirect) | **FIX** — no test date or staleness; no target range |
| PowerPairCard | Watts + Amps pair | — | Value → Telemetry | **CUT?** — not imported anywhere (dead code) |

Device personality (`lib/devicePersonality.ts`): heater coral, pump/ATO/wavemaker/chiller/fan cyan,
skimmer violet `#b09fff`, light gold `#ffd264`, reactor/doser/chemistry green; each has a
blob border-radius. Inferred from `device_type` or name regex. **KEEP the idea**; its coral
for heaters collides with "critical" coral.

Control lock: locked by default, local state, resets on navigation. Unlocking reveals
OFF/ON/AUTO on every controllable card.

## 3. Livestock (`/livestock`, `/livestock/:id`)

List:
- Header pills: "3 Fish · 18 Corals · 11 Invertebrates · 9 Need attention" — **BUG**: counts include deceased; "need attention" = every non-healthy status, so deceased count as needing attention (8 deceased + 1 sick).
- Filters: Type, Status (dropdowns; bottom sheet on phone).
- Grid of portrait 3:4 photo cards (2 cols phone → 4 cols desktop): photo or type-tinted icon placeholder, status dot (healthy dot pulses), name, italic species, status text, ×qty. Sorted alphabetically. **FIX**: deceased shown by default and look identical to living; only 4 cards per desktop screen; card corners inconsistent on phone.
- Add form expands inline: name, species (autocomplete from existing species), type, quantity, status, date added, notes, photo.

Detail:
- Back link, Edit (inline form), Delete (trash → "Confirm remove").
- **Hero photo** 16:9 with name/species over a gradient — **KEEP**, the photos are the most beautiful thing in the app.
- Hover photo tools: replace, crop/rotate (`ImageEditModal`), reset to original, remove (no confirm).
- Chips: type, status, quantity, tenure ("7y 11m in tank").
- Notes block, "All photos" filmstrip (item photo + observation photos) → **Lightbox** with arrows/keyboard.
- Observation log: newest first, relative time, optional status change, note, optional photo (crop/reset/remove). "Log observation" form: status (optional), note, photo. Observations cannot be edited or deleted.
- Max width 3xl: large empty margins on desktop.
- Real data: Toadstool Leather marked sick **151 days ago**, no observation since.

Status change side effect: an auto Journal entry "X status changed to deceased" (Observation · Bad · auto).

## 4. Chemistry (`/chemistry`)

Measurements tab:
- Parameter filter; table Date · Parameter · Value (monospace) · Notes · delete (click-twice confirm). Row dividers.
- Add form (inline card): parameter (Alk, Ca, Mg, NO₃, NO₂, PO₄, salinity, pH, NH₃, silicate), **test kit** (None, Hanna HI772, Red Sea Alk Pro, Salifert KH — converts kit reading to value), value, tested-at, notes. **One parameter per submit** — a 5-parameter test is 5 forms.
- No latest-values summary, no trend chart, no target ranges on this page (trends only via Log › Telemetry › Tests).
- Last test: Jun 4 (5 months ago); nothing on the page says so.

Dosing tab:
- Due Now (overdue doses, "Log Dose"), Dosing Schedules (product, amount, frequency, next due, last, log-now ✓, delete ✗ no confirm), Recent Doses, collapsible Product Catalog (brand, name, type, unit; edit; delete no confirm).
- Schedules can't be edited. Real data: both daily doses last logged 179 days ago.

## 5. Maintenance (`/maintenance`)

Tasks tab:
- Due Now (overdue + due today, computed client-side), "Mark Done" → optional note → "Confirm Done".
- Recurring Tasks: every task again with label (Overdue / Due today / date), description, frequency, last done, Done / Edit / Delete (no confirm).
- Real data: all 7 tasks 4–6 months overdue; every row the same red; no "how late".
- Phone: "Mark Done" wraps to two lines; card-inside-card padding.

Control tab (outlet control lives here):
- "Programs synced 69d ago" + Sync Programs.
- Table (desktop) / cards (phone) of **every Apex output**: real outlets plus alarms (`Alarm_1_2`, `Email Alm_ I5`), serial (`Blu LED_3_5`), variable (Kessil Color/Power), virtual. Raw Apex names, not display names. State label, type, OFF/ON/AUTO buttons (**not** behind the Dashboard lock), PROG.
- **ProgramView** — expands the Apex program, numbered lines (e.g. Heater: Fallback OFF; If Tmp < 78.0 Then ON; If Tmp > 79.0 Then OFF; …). **KEEP**, genuinely useful.
- Recent Events: outlet audit log table (time, outlet, from→to, initiated_by chip ui/cli/mcp/api), limit selector.

## 6. Log (`/history` — nav "Log", title "Aquatic History")

- **Journal** (default tab): entries as cards: sentiment dot, title, category chip (Observation/Maintenance/Event/Milestone), sentiment chip, "auto" chip, relative time, "Show details". Filters: category, sentiment. New Entry: template picker, category, sentiment, title, details. Edit/delete with inline confirm.
  - Real data: dominated by auto entries — "Fed the fish (Feed Mode A)" every few days, status changes, daily check-in answers. Human notes are buried.
- **Telemetry**: probe multi-select (30 raw names incl. `Sw1`–`Sw6`, `Swx1_1`, amps channels), Tests multi-select, time range picker (presets + custom), interval (Auto…), **uPlot** chart with fill, crosshair readout, Export CSV per probe. **KEEP** the chart; no target bands, alert thresholds or event annotations.
- **Full Timeline**: audit events (auth, observation, outlet, …) with kind filter and "Raw payload ↓". Developer-flavored.

## 7. Alerts (`/alerts`)

- Rules table (cards on phone): probe (raw name e.g. "Tmp"), condition, threshold, severity chip, enabled toggle, edit, delete (inline confirm). Disabled rows look identical to enabled.
- **AlertRuleForm** is a **modal** (other forms in the app are inline panels): probe select (defaults to `ATOA`), condition segmented, threshold(s), severity, cooldown, enabled.
- Recent Events below the rules: probe, severity, Active/Cleared, peak, fired/cleared times.
- Real data: 4 of 13 rules enabled; LEAK, all pH, Return Pump, Skimmer, Gyre rules **off**; a pH warning from Apr 17 still "Active".

## 8. Settings (`/settings`) — 11 tabs, tab not kept in URL

| Tab | Contents |
|---|---|
| Dashboard | Layout builder: drag to reorder (dnd-kit), add item, + separator (inline-renamable), compact/normal toggle, delete |
| Devices | Equipment catalog (12): name, type, brand/model ("Kessil A360WE Tuna Blue"), linked outlet id, probe count, Edit; **Suggest** (auto-detect), Add Device. Includes non-Apex gear (CO₂ scrubber, GFO reactor, RO/DI) |
| Tank | Tank profile: Display 60 gal reef, Sump 20 gal (expandable) |
| Probes | Per-probe display name, unit, **min/max normal, min/max warn** (click to edit). *These drive the status colors and the Dashboard headline* |
| Outlets | Apex outlet → display name, id |
| Agent | AI assistant persona: tone, dosing product line, net volume override, custom guardrails, skills toggles |
| Tokens | API tokens list + create (shows "No API tokens" to a read-only token instead of an access error) |
| Notifications | ntfy targets: label, URL (shown in full, includes secret path), status, Send Test, Add |
| Backup | Run Backup Now; **"No backups yet"** |
| Log | System log stream, filter All/API/Poller, Refresh |
| System | Poller status, controller serial/firmware, DB sizes (telemetry 521 MB), event-bus health, Interface › Floating bubbles toggle |

## 9. Bugs and logic problems found

1. **Dashboard value taps open an empty chart.** Cards navigate to `/history?tab=telemetry&probe=Tmp`; `HistoryPage` owns the selection state (initialised `[]`) and ignores the `probe` param → "Select a probe…".
2. **Two unrelated threshold systems.** Probe config (Settings › Probes min/max) sets status colors and the Dashboard headline; alert rules (Alerts) send notifications. pH is "critical" on the Dashboard while every pH rule is disabled.
3. **Alert events never clear** when the rule is disabled (Apr 17 pH warning still Active).
4. **"Due" computed twice**: Dashboard (`/api/tasks/due`) says 6 tasks; Maintenance (client-side) lists 7.
5. **Livestock counts include deceased**; "Need attention" counts deceased.
6. **Tasks linked to dead livestock keep firing** (Pink Frogspawn Recession Check, daily).
7. **Settings tab not in URL**; refresh always lands on Dashboard tab.
8. **Unconfirmed destructive actions**: task delete, dosing schedule delete, product delete, livestock photo remove, observation photo remove.
9. Phone task inputs `text-sm` → iOS zoom on focus.
10. Empty-state shown for permission errors (Tokens tab).
11. Notification target URL displays its secret path.

## 10. Cross-cutting inconsistencies

- **Forms**: inline expanding card (Livestock, Measurements, Tasks, Dosing, Journal, Observation) vs **modal** (Alert rule, image editor) vs **bottom sheet** (phone filters). Label styles differ (`text-xs uppercase` vs `text-xs uppercase tracking-wider` vs sentence case).
- **Confirmations**: 4 patterns (inline Confirm/Cancel, click-twice, "Confirm remove", "Confirm Done") plus none.
- **Buttons**: ~10 radius/size/weight combinations; primary actions are sometimes solid cyan, sometimes `primary/15` tint.
- **Section headers**: `text-sm uppercase`, `text-lg font-semibold`, `text-xs` label — all in use.
- **Widths**: `max-w-3xl` (Livestock detail), `6xl` (Livestock, Chemistry, Maintenance), `7xl` (Dashboard, Log, Alerts, Settings).
- **Names**: Log / Aquatic History / Journal / Settings › Log; Maintenance contains Control; raw Apex names (`Tmp`, `ATOA`, `Sw1`) leak into Alerts, Telemetry, Control.
- **Status vocabulary**: AUTO OFF / INACTIVE / OFF / Standby for similar states.
- **Color**: 155 hard-coded hexes, 125 inline `style=` props (30 in `Layout.tsx`), amber used in 16 files with no token, per-page accents outside the palette, heater coral = critical coral.
- **Time**: almost everything relative ("151d ago") with no absolute date on hover.

## 11. Signature visuals to preserve

These are what make Symbiont feel like Symbiont; the rebuild should make them more
consistent, not remove them.

1. **Sparklines** on device/probe cards (luminous trail, area fill).
2. **LightCycleChart** — multi-channel light intensity curves with legend.
3. **Device personality** — per-equipment icon, blob-shaped icon container, tint.
4. **Livestock photography** — hero images, photo grid, filmstrip, lightbox.
5. **Status-driven ambient light** — caustics re-tinted by tank health.
6. **Bioluminescent pulse** on healthy/live indicators.
7. **Sidebar silhouettes** per page.
8. **Telemetry chart** (uPlot) with crosshair readout.
9. **ProgramView** — readable Apex program per outlet.
10. Bubbles (optional).

## 12. What the first redesign mockups missed

- Hard-coded the Dashboard instead of rendering the user's layout (sections, compact/normal, any mix of card types).
- Dropped light cycle charts, device personality, blob icons, sidebar art, caustics, and replaced photos with placeholders.
- Invented a 6-item nav without accounting for Control's audit log, ProgramView, Telemetry, Timeline, and the 11 Settings tabs.
- No plan for the threshold split (§9.2), which is a product decision, not a styling one.
