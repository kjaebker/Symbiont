# Design System Specification

## 1. Overview & Creative North Star
The Creative North Star for this system is **"The Abyssal Laboratory."** 

This is not a standard utility dashboard; it is a premium, immersive interface that balances the clinical precision of marine science with the organic, rhythmic beauty of a living reef. To move beyond a "template" look, the system leverages high-contrast typography, intentional asymmetry, and deep-sea layering. 

The goal is to make the user feel like they are peering through a reinforced glass portal into a high-end aquatic environment. We achieve this by breaking the traditional grid—using overlapping glass modules, floating "luminous" data points, and a complete rejection of rigid structural lines.

---

## 2. Color & Surface Architecture
The palette is rooted in the darkness of the midnight zone, using vibrant neon accents to simulate bioluminescent life.

### The "No-Line" Rule
**Explicit Instruction:** Prohibit the use of 1px solid borders for sectioning or containment. Boundaries must be defined strictly through background color shifts or subtle tonal transitions. For example, a `surface-container-low` module should sit on a `surface` background to create a soft, natural edge. 

### Surface Hierarchy & Nesting
Treat the UI as a series of physical layers—stacked sheets of frosted glass submerged in deep water.
- **Base Layer:** `surface` (#070d1f) for the global background.
- **Sectioning:** Use `surface-container-low` (#0c1326) for large layout areas.
- **Primary Modules:** Use `surface-container` (#11192e) for main interactive cards.
- **Interactive Details:** Use `surface-container-high` (#171f36) or `highest` (#1c253e) for nested elements like input fields or buttons within a card.

### The "Glass & Gradient" Rule
To achieve "The Abyssal Laboratory" aesthetic, floating elements must utilize **Glassmorphism**. 
- **Effect:** Apply `surface-variant` (#1c253e) at 60% opacity with a `backdrop-blur` of 12px–20px. 
- **Signature Textures:** For primary actions or high-level status headers, use a linear gradient: `primary` (#3adffa) to `primary-container` (#00cbe6) at a 135-degree angle. This mimics the shimmering refraction of light through water.

---

## 3. Typography
We use **Manrope** exclusively. Its geometric yet rounded construction bridges the gap between technical precision and approachable warmth.

- **Display (Large/Medium/Small):** Reserved for high-impact environmental data (e.g., Temperature, pH). Use `display-lg` (3.5rem) to create a bold "Editorial" focal point.
- **Headlines:** Used for section titles. Pair `headline-sm` (1.5rem) with generous letter-spacing (-0.02em) to command authority.
- **Body & Labels:** Use `body-md` (0.875rem) for general telemetry data. Use `label-md` (0.75rem) in `all-caps` with 0.05em tracking for technical metadata to evoke a laboratory instrument feel.

**Hierarchy Note:** Always prioritize the "Current State" value (Display) over the "Category Label" (Label). The data is the hero; the UI is the vessel.

---

## 4. Elevation & Depth
In an underwater environment, there are no hard shadows. Depth is fluid and ambient.

- **The Layering Principle:** Stack `surface-container-lowest` (#000000) modules inside `surface-container-low` (#0c1326) areas to create "wells" of depth. Use `surface-container-highest` (#1c253e) to create "floating" highlights.
- **Ambient Shadows:** Shadows must be extra-diffused. Use a blur of 40px and a spread of -10px with the color `surface-container-lowest` at 40% opacity. This creates a "glow" of darkness rather than a drop shadow.
- **The Ghost Border:** If a boundary is required for accessibility, use a "Ghost Border" of `outline-variant` (#41475b) at 15% opacity. Never use 100% opaque lines.
- **Luminous Trails:** Sparklines and charts should use the `primary` (#3adffa) color with a drop-shadow of the same color (opacity 30%, blur 8px) to create a glowing "trail" effect.

---

## 5. Components

### Buttons & Interaction
- **Primary Button:** Gradient of `primary` to `primary-container`. `xl` (3rem) roundedness. No border. Soft glow on hover.
- **Secondary Button:** `surface-container-high` background. `xl` roundedness. Text in `primary`.
- **Tertiary/Ghost:** `on-surface` text with no background. Interaction state is indicated by a subtle `primary` glow pulse.

### Selection & Control
- **Chips:** Use `secondary-container` (#006d36) for active "Kelp" filter states. All chips must use `full` (9999px) roundedness.
- **Checkboxes/Radios:** Never use standard squares. Use custom circular toggles that "fill" with a `tertiary` (#ff8796) "Coral Pink" glow when active.
- **Input Fields:** `surface-container-highest` backgrounds. Use a `px` bottom border of `primary` only when focused, simulating a light beam.

### Cards & Lists
- **Rule:** **Strictly forbid divider lines.** Use `1.4rem` (Spacing-4) of vertical white space or a subtle shift to `surface-container-low` to separate items.
- **Luminous Status:** Every card should feature a "Bioluminescent Indicator"—a 4px circle of `secondary` (#6dfe9c) that uses a CSS `pulse` animation (opacity 1.0 to 0.4) to indicate a live, healthy system.

### Custom Component: The "Hydro-Graph"
A specialized telemetry card using glassmorphism. It features a background "water-wave" SVG masked within the card, moving slowly at 0.05fps, colored with a gradient of `surface-container` to `surface-bright`.

---

## 6. Do’s and Don’ts

### Do
- **Do** use `xl` (3rem) roundedness for large containers to maintain the "organic" feel.
- **Do** allow elements to overlap slightly (e.g., an icon breaking the edge of a card) to create a high-end, bespoke layout.
- **Do** use `tertiary` (#ff8796) sparingly for critical alerts; it should pop like a rare coral against the deep blue.

### Don't
- **Don't** use pure white (#FFFFFF). Always use `on-surface` (#dfe4fe) to maintain the moody, immersive atmosphere.
- **Don't** use standard 1px borders or grid lines. It breaks the "fluid" immersion.
- **Don't** use abrupt transitions. All hover and active states should have at least a 300ms ease-in-out duration to mimic the resistance of water.