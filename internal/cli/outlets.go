package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewOutletsCmd returns the "outlets" command group.
func NewOutletsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outlets",
		Short: "View and control outlets",
	}
	cmd.AddCommand(newOutletsListCmd(client))
	cmd.AddCommand(newOutletsSetCmd(client))
	cmd.AddCommand(newOutletsEventsCmd(client))
	return cmd
}

func newOutletsListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List current outlet states",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Outlets []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"display_name"`
					State       string `json:"state"`
					Type        string `json:"type"`
					Intensity   int    `json:"intensity"`
				} `json:"outlets"`
			}
			if err := client.Get(cmd.Context(), "/api/outlets", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			headers := []string{"ID", "NAME", "STATE", "TYPE"}
			rows := make([][]string, 0, len(resp.Outlets))
			for _, o := range resp.Outlets {
				rows = append(rows, []string{
					o.ID,
					o.DisplayName,
					ColorState(o.State),
					o.Type,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newOutletsSetCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "set <id> <ON|OFF>",
		Short: "Set an outlet state (AUTO must be done via Apex web UI)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			state := strings.ToUpper(args[1])

			if state != "ON" && state != "OFF" {
				return fmt.Errorf("state must be ON or OFF (got %q); AUTO is not supported by the Apex REST API", args[1])
			}

			var resp struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				State    string `json:"state"`
				LoggedAt string `json:"logged_at"`
			}
			body := map[string]string{"state": state}
			if err := client.Put(cmd.Context(), "/api/outlets/"+id, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Outlet %q set to %s\n", resp.Name, ColorState(resp.State))
			return nil
		},
	}
}

func newOutletsEventsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show outlet change event log",
		RunE: func(cmd *cobra.Command, args []string) error {
			outletID, _ := cmd.Flags().GetString("outlet-id")
			limit, _ := cmd.Flags().GetString("limit")

			path := "/api/outlets/events?"
			if outletID != "" {
				path += "outlet_id=" + outletID + "&"
			}
			if limit != "" {
				path += "limit=" + limit + "&"
			}

			var resp struct {
				Events []struct {
					ID         int64   `json:"id"`
					OutletID   string  `json:"outlet_id"`
					OutletName *string `json:"outlet_name"`
					FromState  *string `json:"from_state"`
					ToState    string  `json:"to_state"`
					InitBy     string  `json:"initiated_by"`
					CreatedAt  string  `json:"created_at"`
				} `json:"events"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			headers := []string{"ID", "OUTLET", "FROM", "TO", "BY", "TIME"}
			rows := make([][]string, 0, len(resp.Events))
			for _, e := range resp.Events {
				name := e.OutletID
				if e.OutletName != nil {
					name = *e.OutletName
				}
				from := "-"
				if e.FromState != nil {
					from = *e.FromState
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", e.ID),
					name,
					from,
					e.ToState,
					e.InitBy,
					formatTimestamp(e.CreatedAt),
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().String("outlet-id", "", "filter by outlet ID")
	cmd.Flags().String("limit", "", "max events to return (default 50)")
	return cmd
}
