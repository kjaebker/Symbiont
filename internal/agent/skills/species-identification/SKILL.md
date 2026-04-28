---
name: species-identification
description: Identify a fish, coral, or invertebrate from a photo or description, then add it to the livestock log with scientific name, type, and care notes pre-filled.
---

Start by calling `get_agent_context` to understand the tank type and current inhabitants.

## Workflow

1. **Load context** — call `get_agent_context`. Note the tank type (reef, FOWLR, mixed), volume, and existing livestock — you'll use this to assess compatibility.

2. **Collect the specimen** — ask the user to either:
   - Share a photo (paste or attach it in the conversation), or
   - Describe the animal in as much detail as they can: colors, shape, pattern, size, behavior, where they bought it

3. **Identify the species** — using your own marine biology knowledge:
   - Provide the most likely **common name** and **scientific name**
   - State your confidence level (high / medium / uncertain)
   - If uncertain, offer 2–3 alternatives and explain what distinguishes them
   - Ask the user to confirm or pick the closest match before proceeding

4. **Compatibility check** — based on the identified species and existing livestock:
   - Flag any known aggression or predation concerns with current inhabitants
   - Flag reef-safe status if the tank is a reef (e.g. some wrasses, angelfish, butterflyfish nip coral)
   - Note minimum tank size requirements vs. current tank volume
   - Note any species that commonly carry disease or need QT

5. **Build the livestock card** — assemble the fields:
   - `name`: common name (e.g. "Flame Angelfish")
   - `species`: scientific name (e.g. "Centropyge loricula")
   - `type`: fish / coral / invertebrate / other
   - `notes`: 2–4 sentences covering care level, reef safety, diet, and any notable husbandry requirements — keep it practical and concise
   - Ask the user: quantity (default 1), and date added (default today)

   Show the assembled card to the user and ask them to confirm or adjust anything before saving.

6. **Save to the log** — call `add_livestock` with the confirmed fields.

7. **Offer to log the acquisition** — offer to call `add_journal_entry` (category: `event`, sentiment: `good`) to record when and where the animal was acquired.

## Tips

- For corals: always include growth form (SPS/LPS/soft), lighting requirement (low/medium/high PAR), and flow preference in the notes.
- For fish: always include temperament (peaceful/semi-aggressive/aggressive) and whether it's a known coral nipper.
- For invertebrates: note any sensitivity to copper medications, as this matters if disease treatment is ever needed.
- If the user isn't sure of the exact variety (e.g. "some kind of Acropora"), use the genus-level scientific name and note it in the card.
