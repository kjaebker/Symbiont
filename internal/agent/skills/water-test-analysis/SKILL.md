---
name: water-test-analysis
description: Analyze reef water test results (Alk/Ca/Mg/NO3/PO4) and produce a safe multi-day dosing plan. Use when the user pastes test numbers.
---

Start by calling the `get_agent_context` MCP tool to load the tank profile, livestock, target parameters, and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`. Note the net volume (gallons), any active alert rule thresholds that reveal target parameters, and the preferred dosing product line.

2. **Collect test values** — if the user has not already provided them, ask for:
   - Alkalinity (dKH)
   - Calcium (ppm)
   - Magnesium (ppm)
   - Nitrate — NO₃ (ppm)
   - Phosphate — PO₄ (ppm)
   - (Optional) Salinity, pH, Temperature

3. **Compare to targets** — use thresholds from the user's alert rules when available. Fall back to typical SPS reef targets:
   - Alk: 8–9.5 dKH | Ca: 420–440 ppm | Mg: 1280–1380 ppm
   - NO₃: 1–10 ppm | PO₄: 0.03–0.1 ppm

4. **Safety checks before dosing**:
   - Correct Magnesium first if < 1200 ppm — low Mg prevents Ca/Alk from staying elevated
   - Never raise Alk > 0.5 dKH per day to avoid spontaneous precipitation
   - Never raise Ca > 20 ppm per day
   - If Alk > 11 dKH AND Ca > 460 ppm simultaneously, precipitation risk is high — recommend a partial water change first
   - Scale all doses to the tank's net volume (from context)

5. **Produce a day-by-day dosing plan** showing:
   - Parameter, current value, target value
   - Product and dose (ml or g, scaled to net volume)
   - Sequencing notes (Mg first; Ca and Alk separated by at least 4 hours to prevent precipitation)
   - Projected value after dose
   - Stop condition (target reached or safe increment met)

6. **Offer to log** — call `add_journal_entry` (category: `maintenance`, sentiment: `neutral`) to record the test results and planned action if the user agrees.
