---
name: water-test-analysis
description: Analyze the most recent water test results already logged in Symbiont and produce a safe multi-day dosing plan. Use after entering test results in the Measurements page.
---

Start by calling `get_agent_context` to load the tank profile, livestock, target parameters, and persona settings.

## Workflow

1. **Load context** — call `get_agent_context`. Note the net volume (gallons), any active alert rule thresholds that reveal target parameters, and the preferred dosing product line.

2. **Fetch stored measurements** — call `get_measurements` to retrieve the logged water chemistry results. Look for the most recent reading of each parameter:
   - Alkalinity (dKH)
   - Calcium (ppm)
   - Magnesium (ppm)
   - Nitrate — NO₃ (ppm)
   - Phosphate — PO₄ (ppm)

   If any key parameter has no recent reading (older than 7 days), ask the user whether they have a fresh value to add via `add_measurement` before proceeding.

3. **Confirm the numbers** — briefly summarize the most recent values and their test dates so the user can confirm they're working from the right data before you do any analysis.

4. **Compare to targets** — use thresholds from the user's alert rules when available. Fall back to typical SPS reef targets:
   - Alk: 8–9.5 dKH | Ca: 420–440 ppm | Mg: 1280–1380 ppm
   - NO₃: 1–10 ppm | PO₄: 0.03–0.1 ppm

5. **Safety checks before dosing**:
   - Correct Magnesium first if < 1200 ppm — low Mg prevents Ca/Alk from staying elevated
   - Never raise Alk > 0.5 dKH per day to avoid spontaneous precipitation
   - Never raise Ca > 20 ppm per day
   - If Alk > 11 dKH AND Ca > 460 ppm simultaneously, precipitation risk is high — recommend a partial water change first
   - Scale all doses to the tank's net volume (from context)

6. **Produce a day-by-day dosing plan** showing:
   - Parameter, current value, target value
   - Product and dose (ml or g, scaled to net volume)
   - Sequencing notes (Mg first; Ca and Alk separated by at least 4 hours to prevent precipitation)
   - Projected value after dose
   - Stop condition (target reached or safe increment met)

7. **Log the analysis** — call `add_journal_entry` (category: `maintenance`, sentiment: `neutral`) to record which parameters were analyzed, their values, and the planned action. This creates a record alongside the measurements already stored.
