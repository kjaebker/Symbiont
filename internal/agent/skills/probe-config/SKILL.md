---
name: probe-config
description: Set min/max normal and min/max warning display bands for each probe using historical data. Use when the dashboard gauges show incorrect ranges or the user wants the UI calibrated to their tank.
---

Start by calling `get_agent_context` to load tank facts and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`. Note the tank type (reef, FOWLR, freshwater) and any target parameters revealed by alert rules.

2. **Fetch current parameters** — call `get_current_parameters` to get all probe names and live values.

3. **Fetch probe configs** — call `list_probe_configs` to see which probes already have display bands configured and what the current values are.

4. **Fetch historical data** — call `get_probe_history` for each probe (period: `30d`). Compute:
   - Observed min and max over the window
   - Typical daily swing (mean difference between daily high and low)
   - Whether the probe is essentially flat (digital inputs, stable dosing probes)

5. **Determine display bands** for each probe using this logic:

   **Normal band** (`min_normal` / `max_normal`) — the range where the probe reads "healthy". Should encompass the probe's typical variation without alarming the user.
   - Start from the observed 5th–95th percentile of historical readings
   - Widen by one daily swing on each side so transient swings don't look out-of-range
   - Snap to known reef targets when available from alert rules or tank profile

   **Warning band** (`min_warning` / `max_warning`) — the outer limit before the gauge turns red. Beyond this, something is wrong.
   - Extend 15–25% beyond the normal band
   - Clamp to physically realistic limits per probe type:
     - Temperature: warn below 74°F (23.3°C) or above 84°F (28.9°C)
     - pH: warn below 7.8 or above 8.6
     - Salinity/Conductivity: warn if > 5% outside normal
     - ORP: warn below 200 mV or above 500 mV

   **Flat probes** (daily swing < 0.5% of range): use a tight normal band centered on the observed mean; keep warning band wide.

6. **Present a calibration plan** in a table:

   | Probe | Current Value | Min Warning | Min Normal | Max Normal | Max Warning | Basis |
   |-------|--------------|------------|-----------|-----------|------------|-------|

   Mark probes that already have correct config as ✓, probes that need updating as ⚠.

7. **Confirm before applying** — ask the user to review the table and approve before writing any changes. Allow the user to adjust individual values before confirming.

8. **Apply approved changes** — for each confirmed probe, call `update_probe_config` with the new band values.

9. **Log the calibration** — call `add_journal_entry` (category: `maintenance`, sentiment: `neutral`) noting how many probes were calibrated and the basis (30-day historical data).
