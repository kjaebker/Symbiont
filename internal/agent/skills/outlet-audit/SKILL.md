---
name: outlet-audit
description: Review outlet event log and current outlet states for anomalies, unexpected cycling, or offline critical equipment.
---

Start by calling `get_agent_context` to load the tank profile and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`.

2. **Fetch current state** — call `get_outlet_states` to see the current state of all outlets (ON/OFF/AUTO/AON/AOF/TBL).

3. **Fetch event log** — call `get_outlet_event_log` to get recent events (48–72 hours minimum). Look for:
   - Outlets that toggled state frequently (rapid cycling = equipment fault, thermostat hunting, or Apex program issue)
   - Critical outlets that are OFF when they should be ON
   - Manual overrides that may have been forgotten

4. **Fetch devices** — call `get_devices` to map outlet IDs to human-readable device names. This makes the report actionable.

5. **Flag anomalies**:
   - **Rapid cycling** — same outlet toggling more than 3 times in an hour: possible thermostat hunting, faulty probe, or GFCI tripping
   - **Extended OFF time** for critical equipment (return pump, heater, skimmer) — immediate attention required
   - **State mismatch** — outlet is in AUTO mode but the Watts probe (if linked) shows unexpectedly low draw
   - **Forgotten manual overrides** — outlets set to manual ON or OFF that should be in AUTO

6. **Produce audit report** in this format:

   ### All Outlets
   | Outlet | Device | Current State | Last Change | Notes |
   |--------|--------|---------------|-------------|-------|

   ### Flagged Items
   List only outlets with anomalies, with severity:
   - ⚠ Warning — unusual but not immediately dangerous
   - 🔴 Critical — potential equipment failure or livestock risk

7. **Offer to log** — call `add_journal_entry` (category: `observation`, sentiment based on findings: `good` if clean, `neutral` if minor issues, `bad` if critical items found) to record the audit results.
