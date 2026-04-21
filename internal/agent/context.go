package agent

import (
	"fmt"
	"strings"

	"github.com/kjaebker/symbiont/internal/db"
)

// ContextInput bundles all DB data needed to assemble the agent context block.
type ContextInput struct {
	Settings         *db.AgentSettings
	Display          *db.TankProfile // may be nil
	Sump             *db.TankProfile // may be nil
	Livestock        []db.LivestockItem
	AlertRules       []db.AlertRule
	Skills           []Skill // enabled skills to list under Available Workflows
	DosingSchedules  []db.DosingSchedule
	DueItems         []db.DueItem
}

// BuildContext assembles a system-prompt context block as a markdown string.
// The output is designed to be pasted into a Claude.ai Project's custom
// instructions, prepended to a Claude Code session, or returned via MCP.
func BuildContext(in ContextInput) string {
	var b strings.Builder

	// --- Identity ---
	b.WriteString("# Symbiont — AI Assistant Context\n\n")
	tone := "analytical"
	if in.Settings != nil && in.Settings.Tone != "" {
		tone = in.Settings.Tone
	}
	fmt.Fprintf(&b, "You are a knowledgeable reef aquarium assistant. Tone: **%s**.\n", tone)
	b.WriteString("Always prioritize the safety of the livestock. Flag any dosing step that carries risk.\n\n")

	// --- Tank facts ---
	b.WriteString("## Tank\n\n")
	vol := netVolume(in)
	if vol > 0 {
		fmt.Fprintf(&b, "- **Net volume:** %.1f gallons\n", vol)
	}
	writeProfileLine(&b, "Display tank", in.Display)
	writeProfileLine(&b, "Sump", in.Sump)
	b.WriteString("\n")

	// --- Livestock ---
	b.WriteString("## Livestock\n\n")
	if len(in.Livestock) == 0 {
		b.WriteString("No livestock recorded.\n\n")
	} else {
		for _, l := range in.Livestock {
			species := ""
			if l.Species != nil {
				species = " (" + *l.Species + ")"
			}
			fmt.Fprintf(&b, "- %s%s — %s, qty %d, status: %s\n",
				l.Name, species, l.Type, l.Quantity, l.Status)
		}
		b.WriteString("\n")
	}

	// --- Target parameters derived from range alert rules ---
	b.WriteString("## Target Parameters\n\n")
	rangeRules := make([]db.AlertRule, 0)
	for _, r := range in.AlertRules {
		if r.Enabled && r.Condition == "outside_range" {
			rangeRules = append(rangeRules, r)
		}
	}
	if len(rangeRules) == 0 {
		b.WriteString("No range alert rules configured (targets unknown).\n\n")
	} else {
		for _, r := range rangeRules {
			lo, hi := "--", "--"
			if r.ThresholdLow != nil {
				lo = fmt.Sprintf("%.4g", *r.ThresholdLow)
			}
			if r.ThresholdHigh != nil {
				hi = fmt.Sprintf("%.4g", *r.ThresholdHigh)
			}
			fmt.Fprintf(&b, "- **%s:** %s – %s\n", r.ProbeName, lo, hi)
		}
		b.WriteString("\n")
	}

	// --- Dosing schedule ---
	b.WriteString("## Dosing\n\n")
	productLine := "generic"
	if in.Settings != nil && in.Settings.DosingProductLine != "" {
		productLine = in.Settings.DosingProductLine
	}
	fmt.Fprintf(&b, "- **Product line:** %s\n", productLine)
	if len(in.DosingSchedules) == 0 {
		b.WriteString("- No dosing schedules configured.\n")
	} else {
		for _, ds := range in.DosingSchedules {
			if !ds.Enabled {
				continue
			}
			brand, name, unit := "", "", ""
			if ds.ProductBrand != nil {
				brand = *ds.ProductBrand
			}
			if ds.ProductName != nil {
				name = *ds.ProductName
			}
			if ds.ProductUnit != nil {
				unit = *ds.ProductUnit
			}
			last := "never"
			if ds.LastCompletedAt != nil {
				last = ds.LastCompletedAt.Format("2006-01-02")
			}
			next := "not scheduled"
			if ds.NextDueAt != nil {
				next = ds.NextDueAt.Format("2006-01-02")
			}
			fmt.Fprintf(&b, "- **%s %s:** %.4g %s, %s — last: %s, next: %s\n",
				brand, name, ds.Amount, unit, ds.Frequency, last, next)
		}
	}
	b.WriteString("\n")

	// --- Due/overdue tasks ---
	if len(in.DueItems) > 0 {
		b.WriteString("## Tasks Due Now\n\n")
		for _, item := range in.DueItems {
			status := "due"
			if item.IsOverdue {
				status = "OVERDUE"
			}
			fmt.Fprintf(&b, "- [%s] **%s** — %s\n", status, item.Label, item.Detail)
		}
		b.WriteString("\n")
	}

	// --- Custom guardrails (user-supplied freetext appended verbatim) ---
	if in.Settings != nil && in.Settings.CustomGuardrails != nil && *in.Settings.CustomGuardrails != "" {
		b.WriteString("## Custom Guardrails\n\n")
		b.WriteString(*in.Settings.CustomGuardrails)
		b.WriteString("\n\n")
	}

	// --- Available skills ---
	if len(in.Skills) > 0 {
		b.WriteString("## Available Workflows\n\n")
		b.WriteString("The following Symbiont skills are installed and may be invoked by name:\n\n")
		for _, s := range in.Skills {
			fmt.Fprintf(&b, "- **%s** — %s\n", s.Name, s.Description)
		}
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// netVolume returns the effective system volume: the settings override if set,
// otherwise the sum of display and sump tank volumes.
func netVolume(in ContextInput) float64 {
	if in.Settings != nil && in.Settings.NetVolumeGallons != nil {
		return *in.Settings.NetVolumeGallons
	}
	var sum float64
	if in.Display != nil && in.Display.VolumeGallons != nil {
		sum += *in.Display.VolumeGallons
	}
	if in.Sump != nil && in.Sump.VolumeGallons != nil {
		sum += *in.Sump.VolumeGallons
	}
	return sum
}

func writeProfileLine(b *strings.Builder, label string, p *db.TankProfile) {
	if p == nil {
		return
	}
	var parts []string
	if p.VolumeGallons != nil {
		parts = append(parts, fmt.Sprintf("%.1f gal", *p.VolumeGallons))
	}
	if p.LengthIn != nil && p.WidthIn != nil && p.HeightIn != nil {
		parts = append(parts, fmt.Sprintf(`%.0f"×%.0f"×%.0f"`, *p.LengthIn, *p.WidthIn, *p.HeightIn))
	}
	if p.TankType != nil {
		parts = append(parts, *p.TankType)
	}
	if len(parts) == 0 {
		return
	}
	fmt.Fprintf(b, "- **%s:** %s\n", label, strings.Join(parts, ", "))
}
