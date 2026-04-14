---
name: parameter-trend-review
description: Pull probe history for the last 7 and 30 days, identify drift and instability, and recommend corrective action.
---

Start by calling `get_agent_context` to load the tank profile, target parameters, and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`. Note the target parameter ranges (from alert rule thresholds) and net volume.

2. **Fetch current parameters** — call `get_current_parameters` to get a snapshot of all current probe readings and their status.

3. **Fetch probe history** — call `get_probe_history` for each key probe:
   - 7-day window: `from` = 7 days ago, `interval` = `1h`
   - 30-day window: `from` = 30 days ago, `interval` = `6h`

   Prioritize probes with active alert rules, then: temperature, salinity, pH, ORP, and power probes for critical equipment.

4. **Analyze each probe**:
   - **Trend:** is the value drifting up or down over 7 days? Calculate slope (first vs. last tercile average).
   - **Stability:** what is the daily peak-to-trough swing? Temperature swings > 2°F, pH swings > 0.3 units, or salinity swings > 0.5 ppt are problematic.
   - **Anomalies:** any sudden spikes or drops that correlate with outlet events?
   - **Compliance:** is the current value inside the alert rule range?

5. **Produce a trend report** — a table for each probe:

   | Probe | Current | 7-day range | 30-day range | Trend | Status |
   |-------|---------|-------------|--------------|-------|--------|
   | Tmp   | 78.2°F  | 77.8–78.6   | 77.5–79.1    | stable | ✅ |

   Status key: ✅ stable | ⚠ drift detected | 🔴 out of range | ❓ insufficient data

6. **Recommendations** — for any probe flagged with drift or out-of-range:
   - Temperature drift → check heater/chiller calibration, inspect HVAC
   - pH declining trend → check CO₂ buildup, evaluate refugium lighting schedule
   - Salinity drift → check ATO reservoir fill level and top-off rate
   - ORP declining → check skimmer, evaluate ozone if applicable

7. **Offer to log** — call `add_journal_entry` (category: `observation`) summarizing the trend review findings and any recommended actions.
