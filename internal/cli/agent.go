package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// NewAgentCmd returns the "agent" command group.
func NewAgentCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agent persona settings and context",
	}
	cmd.AddCommand(newAgentContextCmd(client))
	cmd.AddCommand(newAgentSettingsCmd(client))
	return cmd
}

// newAgentContextCmd prints the assembled agent context markdown.
//
//	symbiont agent context
func newAgentContextCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Print the assembled AI assistant context (markdown)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Context string `json:"context"`
			}
			if err := client.Get(cmd.Context(), "/api/agent/context", &resp); err != nil {
				return err
			}
			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}
			fmt.Print(resp.Context)
			return nil
		},
	}
}

// newAgentSettingsCmd shows agent settings and optionally updates them.
//
//	symbiont agent settings
//	symbiont agent settings --set tone=casual
//	symbiont agent settings --set dosing_product_line=brs_pharma
//	symbiont agent settings --set net_volume_gallons=120
func newAgentSettingsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Show or update agent persona settings",
		Long: `Show current agent settings, or update individual fields with --set key=value.

Valid keys:
  tone                 analytical | casual | terse
  dosing_product_line  brs_pharma | red_sea | tropic_marin | generic | none
  net_volume_gallons   number (overrides sum of tank profile volumes)
  custom_guardrails    free-text appended to every context block`,
		RunE: func(cmd *cobra.Command, args []string) error {
			setPairs, _ := cmd.Flags().GetStringArray("set")

			if len(setPairs) > 0 {
				return agentSettingsUpdate(cmd, client, setPairs)
			}
			return agentSettingsShow(cmd, client)
		},
	}
	cmd.Flags().StringArray("set", nil, "set key=value (repeatable)")
	return cmd
}

type agentSettingsJSON struct {
	Tone              string   `json:"tone"`
	DosingProductLine string   `json:"dosing_product_line"`
	NetVolumeGallons  *float64 `json:"net_volume_gallons"`
	CustomGuardrails  *string  `json:"custom_guardrails"`
	EnabledSkills     []string `json:"enabled_skills"`
	UpdatedAt         string   `json:"updated_at"`
}

func agentSettingsShow(cmd *cobra.Command, client *APIClient) error {
	var s agentSettingsJSON
	if err := client.Get(cmd.Context(), "/api/agent/settings", &s); err != nil {
		return err
	}
	if IsJSON(cmd) {
		PrintJSON(s)
		return nil
	}

	pstr := func(v *string) string {
		if v == nil || *v == "" {
			return "(not set)"
		}
		return *v
	}
	pfloat := func(v *float64) string {
		if v == nil {
			return "(derived from tank profile)"
		}
		return fmt.Sprintf("%.1f", *v)
	}

	fmt.Printf("Tone:                 %s\n", s.Tone)
	fmt.Printf("Dosing product line:  %s\n", s.DosingProductLine)
	fmt.Printf("Net volume (gallons): %s\n", pfloat(s.NetVolumeGallons))
	fmt.Printf("Custom guardrails:    %s\n", pstr(s.CustomGuardrails))
	if len(s.EnabledSkills) == 0 {
		fmt.Println("Enabled skills:       (all)")
	} else {
		fmt.Printf("Enabled skills:       %s\n", strings.Join(s.EnabledSkills, ", "))
	}
	fmt.Printf("Updated at:           %s\n", s.UpdatedAt)
	return nil
}

func agentSettingsUpdate(cmd *cobra.Command, client *APIClient, setPairs []string) error {
	body := map[string]any{}
	for _, pair := range setPairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("invalid --set format %q: expected key=value", pair)
		}
		switch k {
		case "tone", "dosing_product_line", "custom_guardrails":
			body[k] = v
		case "net_volume_gallons":
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("net_volume_gallons must be a number, got %q", v)
			}
			body[k] = f
		default:
			return fmt.Errorf("unknown setting %q; valid keys: tone, dosing_product_line, net_volume_gallons, custom_guardrails", k)
		}
	}

	var s agentSettingsJSON
	if err := client.Put(cmd.Context(), "/api/agent/settings", body, &s); err != nil {
		return err
	}
	if IsJSON(cmd) {
		PrintJSON(s)
		return nil
	}
	fmt.Println("Agent settings updated.")
	return nil
}
