---
name: new-addition-acclimation
description: Guide the user through safe drip acclimation and quarantine protocols when adding a new fish, coral, or invertebrate.
---

Start by calling `get_agent_context` to load the tank profile and existing livestock.

## Workflow

1. **Load context** — call `get_agent_context`. Note the tank type, volume, and existing livestock to assess compatibility.

2. **Identify the new addition** — ask the user:
   - What species / type? (fish, coral, invertebrate, other)
   - Where did it come from? (LFS, online vendor, frag swap, wild-caught)
   - What are the bag parameters if known? (temp, salinity)

3. **Compatibility check** — based on existing livestock and tank type:
   - Flag any known aggression issues with current inhabitants
   - Flag reef-safe concerns for fish if tank is reef; note if coral is not appropriate for FOWLR chemistry
   - Recommend QT if source is unknown, wild-caught, or the system is disease-naive

4. **Drip acclimation protocol** (standard for fish and sensitive inverts):
   - Float bag in sump or bucket for 15 minutes to equalize temperature
   - Open bag, place water + animal in clean container
   - Drip at 2–4 drops/second using airline tubing with a loose knot
   - Target: double the water volume over 60–90 minutes
   - Never add bag water to the display tank
   - Net transfer the animal to the display tank or QT

5. **Coral / frag protocol**:
   - Temperature float 10–15 min
   - Dip in coral dip solution (iodine-based or CoralRx) for 5–10 min per product instructions
   - Rinse in tank water before placing
   - Place in low flow, low light area initially and observe polyp extension over 24–48 hours

6. **Quarantine guidance**:
   - Recommend 4–6 week QT minimum for all new fish
   - Prophylactic treatment options: tank transfer method (TTM) for ich, PraziPro for flukes
   - QT is generally not required for coral or most inverts, but a coral dip is always recommended

7. **Log the addition** — offer to:
   - Call `add_livestock` to record the new animal in the system
   - Call `add_journal_entry` (category: `event`, sentiment: `good`) to log the arrival date and source
