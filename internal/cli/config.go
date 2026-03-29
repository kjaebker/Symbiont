package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewConfigCmd returns the "config" command group.
func NewConfigCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage probe and outlet configuration",
	}
	probesCmd := &cobra.Command{
		Use:   "probes",
		Short: "Manage probe display configuration",
	}
	probesCmd.AddCommand(newConfigProbesListCmd(client))
	probesCmd.AddCommand(newConfigProbesUpdateCmd(client))

	outletsCmd := &cobra.Command{
		Use:   "outlets",
		Short: "Manage outlet display configuration",
	}
	outletsCmd.AddCommand(newConfigOutletsListCmd(client))
	outletsCmd.AddCommand(newConfigOutletsUpdateCmd(client))

	cmd.AddCommand(probesCmd)
	cmd.AddCommand(outletsCmd)
	return cmd
}

func newConfigProbesListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List probe configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Configs []struct {
					ProbeName   string   `json:"probe_name"`
					DisplayName *string  `json:"display_name"`
					UnitOverride *string `json:"unit_override"`
					MinNormal   *float64 `json:"min_normal"`
					MaxNormal   *float64 `json:"max_normal"`
					MinWarning  *float64 `json:"min_warning"`
					MaxWarning  *float64 `json:"max_warning"`
				} `json:"configs"`
			}
			if err := client.Get(cmd.Context(), "/api/config/probes", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Configs) == 0 {
				fmt.Println("No probe configurations.")
				return nil
			}

			headers := []string{"PROBE", "DISPLAY NAME", "UNIT", "NORMAL RANGE", "WARNING RANGE"}
			rows := make([][]string, 0, len(resp.Configs))
			for _, c := range resp.Configs {
				displayName := "-"
				if c.DisplayName != nil {
					displayName = *c.DisplayName
				}
				unit := "-"
				if c.UnitOverride != nil {
					unit = *c.UnitOverride
				}
				normalRange := formatOptionalRange(c.MinNormal, c.MaxNormal)
				warnRange := formatOptionalRange(c.MinWarning, c.MaxWarning)
				rows = append(rows, []string{c.ProbeName, displayName, unit, normalRange, warnRange})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newConfigProbesUpdateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update probe display configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			body := make(map[string]any)

			if cmd.Flags().Changed("display-name") {
				v, _ := cmd.Flags().GetString("display-name")
				body["display_name"] = v
			}
			if cmd.Flags().Changed("unit") {
				v, _ := cmd.Flags().GetString("unit")
				body["unit_override"] = v
			}
			if cmd.Flags().Changed("min-normal") {
				v, _ := cmd.Flags().GetFloat64("min-normal")
				body["min_normal"] = v
			}
			if cmd.Flags().Changed("max-normal") {
				v, _ := cmd.Flags().GetFloat64("max-normal")
				body["max_normal"] = v
			}
			if cmd.Flags().Changed("min-warning") {
				v, _ := cmd.Flags().GetFloat64("min-warning")
				body["min_warning"] = v
			}
			if cmd.Flags().Changed("max-warning") {
				v, _ := cmd.Flags().GetFloat64("max-warning")
				body["max_warning"] = v
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var resp any
			if err := client.Put(cmd.Context(), "/api/config/probes/"+name, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Probe config for %q updated\n", name)
			return nil
		},
	}
	cmd.Flags().String("display-name", "", "display name")
	cmd.Flags().String("unit", "", "unit override")
	cmd.Flags().Float64("min-normal", 0, "minimum normal value")
	cmd.Flags().Float64("max-normal", 0, "maximum normal value")
	cmd.Flags().Float64("min-warning", 0, "minimum warning value")
	cmd.Flags().Float64("max-warning", 0, "maximum warning value")
	return cmd
}

func newConfigOutletsListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List outlet configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Configs []struct {
					OutletID    string  `json:"outlet_id"`
					DisplayName *string `json:"display_name"`
					Icon        *string `json:"icon"`
				} `json:"configs"`
			}
			if err := client.Get(cmd.Context(), "/api/config/outlets", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Configs) == 0 {
				fmt.Println("No outlet configurations.")
				return nil
			}

			headers := []string{"OUTLET ID", "DISPLAY NAME", "ICON"}
			rows := make([][]string, 0, len(resp.Configs))
			for _, c := range resp.Configs {
				displayName := "-"
				if c.DisplayName != nil {
					displayName = *c.DisplayName
				}
				icon := "-"
				if c.Icon != nil {
					icon = *c.Icon
				}
				rows = append(rows, []string{c.OutletID, displayName, icon})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newConfigOutletsUpdateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update outlet display configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]any)

			if cmd.Flags().Changed("display-name") {
				v, _ := cmd.Flags().GetString("display-name")
				body["display_name"] = v
			}
			if cmd.Flags().Changed("icon") {
				v, _ := cmd.Flags().GetString("icon")
				body["icon"] = v
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var resp any
			if err := client.Put(cmd.Context(), "/api/config/outlets/"+id, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Outlet config for %q updated\n", id)
			return nil
		},
	}
	cmd.Flags().String("display-name", "", "display name")
	cmd.Flags().String("icon", "", "icon identifier")
	return cmd
}

func formatOptionalRange(min, max *float64) string {
	if min != nil && max != nil {
		return fmt.Sprintf("%.2f – %.2f", *min, *max)
	}
	if min != nil {
		return fmt.Sprintf(">= %.2f", *min)
	}
	if max != nil {
		return fmt.Sprintf("<= %.2f", *max)
	}
	return "-"
}
