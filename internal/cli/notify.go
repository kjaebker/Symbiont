package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewNotifyCmd returns the "notify" command group.
func NewNotifyCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Manage notification targets",
	}
	cmd.AddCommand(newNotifyListCmd(client))
	cmd.AddCommand(newNotifyCreateCmd(client))
	cmd.AddCommand(newNotifyDeleteCmd(client))
	cmd.AddCommand(newNotifyTestCmd(client))
	return cmd
}

func newNotifyListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notification targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Targets []struct {
					ID      int64  `json:"id"`
					Type    string `json:"type"`
					Label   string `json:"label"`
					Enabled bool   `json:"enabled"`
					Config  string `json:"config"`
				} `json:"targets"`
			}
			if err := client.Get(cmd.Context(), "/api/notifications/targets", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Targets) == 0 {
				fmt.Println("No notification targets configured.")
				return nil
			}

			headers := []string{"ID", "TYPE", "LABEL", "ENABLED", "CONFIG"}
			rows := make([][]string, 0, len(resp.Targets))
			for _, t := range resp.Targets {
				enabled := "yes"
				if !t.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", t.ID),
					t.Type,
					t.Label,
					enabled,
					t.Config,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newNotifyCreateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create or update a notification target",
		RunE: func(cmd *cobra.Command, args []string) error {
			ntype, _ := cmd.Flags().GetString("type")
			label, _ := cmd.Flags().GetString("label")
			config, _ := cmd.Flags().GetString("config")
			enabled, _ := cmd.Flags().GetBool("enabled")

			body := map[string]any{
				"type":    ntype,
				"label":   label,
				"config":  config,
				"enabled": enabled,
			}

			var resp struct {
				ID      int64  `json:"id"`
				Type    string `json:"type"`
				Label   string `json:"label"`
				Enabled bool   `json:"enabled"`
			}
			if err := client.Post(cmd.Context(), "/api/notifications/targets", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Notification target #%d created (%s: %s)\n", resp.ID, resp.Type, resp.Label)
			return nil
		},
	}
	cmd.Flags().String("type", "", "target type (e.g. ntfy) (required)")
	cmd.Flags().String("label", "", "human-readable label (required)")
	cmd.Flags().String("config", "", "target config (e.g. ntfy URL) (required)")
	cmd.Flags().Bool("enabled", true, "enable this target")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("label")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

func newNotifyDeleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a notification target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Delete notification target #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/notifications/targets/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "deleted", "id": id})
				return nil
			}

			fmt.Printf("Notification target #%s deleted\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}

func newNotifyTestCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Send a test notification to all enabled targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Results []struct {
					Label   string `json:"label"`
					Success bool   `json:"success"`
					Error   string `json:"error,omitempty"`
				} `json:"results"`
			}
			if err := client.Post(cmd.Context(), "/api/notifications/test", nil, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			for _, r := range resp.Results {
				if r.Success {
					fmt.Printf("  %s  %s\n", colorGreen+"OK"+colorReset, r.Label)
				} else {
					fmt.Printf("  %s  %s: %s\n", colorRed+"FAIL"+colorReset, r.Label, r.Error)
				}
			}
			return nil
		},
	}
}
