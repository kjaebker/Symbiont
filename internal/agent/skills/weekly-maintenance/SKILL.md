---
name: weekly-maintenance
description: Generate a prioritized weekly maintenance checklist drawing from recent journal entries, active alerts, and outlet anomalies.
---

Start by calling `get_agent_context` to load the tank profile, livestock, and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`.

2. **Fetch recent data** (call in parallel where possible):
   - `get_journal_entries` with `limit=20` to see recent activity
   - `get_alert_events` to see any active or recent alerts
   - `get_outlet_event_log` to spot any unusual power cycling

3. **Build the checklist** — organize tasks into sections:

   ### Water
   - [ ] Test alkalinity, calcium, magnesium
   - [ ] Test nitrate and phosphate
   - [ ] Check salinity (refractometer or probe)
   - (Add urgent items if alert rules have fired recently)

   ### Equipment
   - [ ] Inspect skimmer cup — clean if more than half full
   - [ ] Check return pump flow
   - [ ] Inspect powerhead impellers for coralline buildup
   - [ ] Check media (carbon/GFO) — schedule replacement if due
   - (Add any outlet anomalies found in the event log)

   ### Livestock
   - List each livestock item from context and prompt for a quick visual health check
   - Flag any items with status `sick` or `deceased` in the system

   ### Maintenance log
   - Review recent journal entries and call out anything that needs follow-up

4. **Offer to log** — when the user says they have completed the checklist (or a portion), call `add_journal_entry` (category: `maintenance`) to record what was done.
