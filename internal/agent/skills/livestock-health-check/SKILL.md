---
name: livestock-health-check
description: Walk through each livestock item, collect health observations, and log them via the Symbiont MCP tools.
---

Start by calling `get_agent_context` to understand the tank type and livestock inventory.

## Workflow

1. **Load context** — call `get_agent_context`. Note the tank type (FOWLR, reef, etc.) and livestock count.

2. **Fetch livestock** — call `get_livestock` to get the current inventory with status and species.

3. **Walk through each item** — for each livestock item:
   - Present the name, species, and current recorded status
   - Ask the user: "How does [name] look today?" and prompt for:
     - **Behavior** — eating, hiding, aggression, lethargy
     - **Appearance** — color, fins, spots, mucus, wounds
     - **Overall status** — healthy / watch / sick / deceased
   - If the user reports anything concerning, ask follow-up questions and provide guidance appropriate to the species

4. **Log observations** — for each item where the user provides information, call `add_livestock_observation` with:
   - The livestock ID
   - The reported status
   - A concise note summarizing what the user observed

5. **Flag concerns** — after walking through all items, summarize:
   - Any items with status changes (e.g., now `sick` or `watch`)
   - Any disease patterns (e.g., multiple fish showing spots → possible ich or velvet)
   - Recommended actions (freshwater dip, QT, medication, vet consult)

6. **Offer to log a summary** — call `add_journal_entry` (category: `observation`) to record the overall health check session.
