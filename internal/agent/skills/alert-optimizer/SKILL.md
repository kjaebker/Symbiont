---
name: alert-optimizer
description: Review existing alert rules against historical probe data and suggest a complete, calibrated alert set based on what this tank actually does. Use when the user wants smarter alerts.
---

Start by calling `get_agent_context` to load tank facts, existing alert rules, and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`. Note the existing alert rules (reveals current thresholds and which probes are already covered).

2. **Fetch current parameters** — call `get_current_parameters` to see all probe names and live values. This is your complete list of probes to work with.

3. **Fetch historical data for each probe** — call `get_probe_history` for each probe (period: `7d`, then `30d` if you need more baseline). Compute:
   - Observed min and max over the window
   - Typical daily swing (difference between daily high and daily low, averaged)
   - Whether the probe has meaningful variance or is flat (e.g. a digital input)

4. **Fetch recent alert events** — call `get_alert_events` to see which rules have been firing. Identify:
   - Rules that fire too frequently (noisy — threshold too tight relative to normal swing)
   - Rules that have never fired (may be redundant or misconfigured)
   - Probes with no alert coverage that have shown concerning historical movement

5. **Categorize probes**:
   - **Critical** — Temperature, pH, Salinity/Conductivity: alert on any significant deviation
   - **Husbandry** — Alkalinity/ORP (if present): alert when trending toward unsafe range
   - **Secondary** — Amp/Watt probes: alert on sustained zero draw (equipment offline)
   - **Noise** — Flat or digitally-switched probes: alert rarely or not at all

6. **Produce a recommended alert set** in a table:

   | Probe | Condition | Low Threshold | High Threshold | Severity | Rationale |
   |-------|-----------|--------------|----------------|----------|-----------|

   Base thresholds on historical data:
   - **Normal band**: observed mean ± (2× daily swing), rounded to one decimal
   - **Warning buffer**: add 10–15% beyond normal band before triggering
   - Clamp to realistic reef limits (temp: never below 74°F / 23.3°C or above 82°F / 27.8°C)
   - Use `outside_range` for probes that should stay in a band; `above`/`below` for one-sided risks

7. **Compare to existing rules** — for each recommendation:
   - If a rule already exists and matches: mark as ✓ (keep as-is)
   - If a rule exists but the threshold is off: mark as ⚠ (suggest update)
   - If no rule exists: mark as ✚ (suggest create)
   - If an existing rule is not in the recommendation set: mark as ✗ (suggest delete)

8. **Confirm before making changes** — present the full before/after plan and ask the user to approve before calling any write tools.

9. **Apply approved changes** — for each confirmed change:
   - Create new rules via `create_alert_rule`
   - Update existing rules via `update_alert_rule`
   - Delete removed rules via `delete_alert_rule`

10. **Log the optimization** — call `add_journal_entry` (category: `maintenance`, sentiment: `good`) summarizing how many rules were added, updated, and removed.
