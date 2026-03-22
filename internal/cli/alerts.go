package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// NewAlertsCmd returns the "alerts" command group.
func NewAlertsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Manage alert rules",
	}
	cmd.AddCommand(newAlertsListCmd(client))
	cmd.AddCommand(newAlertsCreateCmd(client))
	cmd.AddCommand(newAlertsUpdateCmd(client))
	cmd.AddCommand(newAlertsDeleteCmd(client))
	return cmd
}

func newAlertsListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List alert rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Rules []struct {
					ID        int64    `json:"id"`
					ProbeName string   `json:"probe_name"`
					Condition string   `json:"condition"`
					Low       *float64 `json:"threshold_low"`
					High      *float64 `json:"threshold_high"`
					Severity  string   `json:"severity"`
					Enabled   bool     `json:"enabled"`
					Cooldown  int      `json:"cooldown_minutes"`
				} `json:"rules"`
			}
			if err := client.Get(cmd.Context(), "/api/alerts", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Rules) == 0 {
				fmt.Println("No alert rules configured.")
				return nil
			}

			headers := []string{"ID", "PROBE", "CONDITION", "THRESHOLD", "SEVERITY", "ENABLED"}
			rows := make([][]string, 0, len(resp.Rules))
			for _, a := range resp.Rules {
				threshold := formatThreshold(a.Low, a.High)
				enabled := "yes"
				if !a.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", a.ID),
					a.ProbeName,
					a.Condition,
					threshold,
					a.Severity,
					enabled,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newAlertsCreateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an alert rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			probe, _ := cmd.Flags().GetString("probe")
			condition, _ := cmd.Flags().GetString("condition")
			severity, _ := cmd.Flags().GetString("severity")
			cooldown, _ := cmd.Flags().GetInt("cooldown")

			body := map[string]any{
				"probe_name":       probe,
				"condition":        condition,
				"severity":         severity,
				"cooldown_minutes": cooldown,
				"enabled":          true,
			}

			if v, _ := cmd.Flags().GetFloat64("low"); cmd.Flags().Changed("low") {
				body["threshold_low"] = v
			}
			if v, _ := cmd.Flags().GetFloat64("high"); cmd.Flags().Changed("high") {
				body["threshold_high"] = v
			}

			var resp struct {
				ID int64 `json:"id"`
			}
			if err := client.Post(cmd.Context(), "/api/alerts", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Alert rule #%d created\n", resp.ID)
			return nil
		},
	}
	cmd.Flags().String("probe", "", "probe name (required)")
	cmd.Flags().String("condition", "", "condition: above, below, outside_range (required)")
	cmd.Flags().Float64("low", 0, "low threshold")
	cmd.Flags().Float64("high", 0, "high threshold")
	cmd.Flags().String("severity", "warning", "severity: warning, critical")
	cmd.Flags().Int("cooldown", 15, "cooldown in minutes")
	_ = cmd.MarkFlagRequired("probe")
	_ = cmd.MarkFlagRequired("condition")
	return cmd
}

func newAlertsUpdateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]any)

			if cmd.Flags().Changed("probe") {
				v, _ := cmd.Flags().GetString("probe")
				body["probe_name"] = v
			}
			if cmd.Flags().Changed("condition") {
				v, _ := cmd.Flags().GetString("condition")
				body["condition"] = v
			}
			if cmd.Flags().Changed("severity") {
				v, _ := cmd.Flags().GetString("severity")
				body["severity"] = v
			}
			if cmd.Flags().Changed("cooldown") {
				v, _ := cmd.Flags().GetInt("cooldown")
				body["cooldown_minutes"] = v
			}
			if cmd.Flags().Changed("low") {
				v, _ := cmd.Flags().GetFloat64("low")
				body["threshold_low"] = v
			}
			if cmd.Flags().Changed("high") {
				v, _ := cmd.Flags().GetFloat64("high")
				body["threshold_high"] = v
			}
			if cmd.Flags().Changed("enabled") {
				v, _ := cmd.Flags().GetBool("enabled")
				body["enabled"] = v
			}

			if err := client.Put(cmd.Context(), "/api/alerts/"+id, body, nil); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "updated", "id": id})
				return nil
			}

			fmt.Printf("Alert rule #%s updated\n", id)
			return nil
		},
	}
	cmd.Flags().String("probe", "", "probe name")
	cmd.Flags().String("condition", "", "condition: above, below, outside_range")
	cmd.Flags().Float64("low", 0, "low threshold")
	cmd.Flags().Float64("high", 0, "high threshold")
	cmd.Flags().String("severity", "", "severity: warning, critical")
	cmd.Flags().Int("cooldown", 0, "cooldown in minutes")
	cmd.Flags().Bool("enabled", true, "enable/disable rule")
	return cmd
}

func newAlertsDeleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an alert rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Delete alert rule #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/alerts/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "deleted", "id": id})
				return nil
			}

			fmt.Printf("Alert rule #%s deleted\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}

func formatThreshold(low, high *float64) string {
	if low != nil && high != nil {
		return strconv.FormatFloat(*low, 'f', 2, 64) + " – " + strconv.FormatFloat(*high, 'f', 2, 64)
	}
	if low != nil {
		return "< " + strconv.FormatFloat(*low, 'f', 2, 64)
	}
	if high != nil {
		return "> " + strconv.FormatFloat(*high, 'f', 2, 64)
	}
	return "-"
}
