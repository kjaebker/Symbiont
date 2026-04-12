package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewFeedCmd returns the "feed" command group.
func NewFeedCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Check or control Apex feed modes",
	}
	cmd.AddCommand(newFeedStatusCmd(client))
	cmd.AddCommand(newFeedSetCmd(client))
	cmd.AddCommand(newFeedCancelCmd(client))
	return cmd
}

func newFeedStatusCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current feed mode status",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Name   int  `json:"name"`
				Active bool `json:"active"`
			}
			if err := client.Get(cmd.Context(), "/api/feed", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			feedName := "Off"
			if resp.Name >= 1 && resp.Name <= 4 {
				feedName = "Feed " + string(rune('A'+resp.Name-1))
			}
			active := "no"
			if resp.Active {
				active = "yes"
			}

			headers := []string{"MODE", "ACTIVE"}
			PrintTable(headers, [][]string{{feedName, active}})
			return nil
		},
	}
}

func newFeedSetCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "set <A|B|C|D>",
		Short: "Activate a feed mode (A, B, C, or D)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedMap := map[string]int{"A": 1, "B": 2, "C": 3, "D": 4}
			name, ok := feedMap[args[0]]
			if !ok {
				return fmt.Errorf("invalid feed mode %q — must be A, B, C, or D", args[0])
			}

			body := map[string]any{"name": name, "active": true}
			var resp struct {
				Name   int  `json:"name"`
				Active bool `json:"active"`
			}
			if err := client.Put(cmd.Context(), "/api/feed", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Feed %s activated\n", args[0])
			return nil
		},
	}
}

func newFeedCancelCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the active feed mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"name": 0, "active": false}
			var resp struct {
				Name   int  `json:"name"`
				Active bool `json:"active"`
			}
			if err := client.Put(cmd.Context(), "/api/feed", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Println("Feed mode cancelled")
			return nil
		},
	}
}

