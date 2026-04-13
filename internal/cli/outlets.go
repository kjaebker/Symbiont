package cli

import (
	"encoding/json"
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
		Use:   "set <id> <ON|OFF|AUTO>",
		Short: "Set an outlet state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			state := strings.ToUpper(args[1])

			if state != "ON" && state != "OFF" && state != "AUTO" {
				return fmt.Errorf("state must be ON, OFF, or AUTO (got %q)", args[1])
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

			path := "/api/events?kind=outlet_changed&"
			if outletID != "" {
				path += "correlation_id=" + outletID + "&"
			}
			if limit != "" {
				path += "limit=" + limit + "&"
			}

			var resp struct {
				Events []struct {
					ID          int64  `json:"id"`
					TS          string `json:"ts"`
					PayloadJSON string `json:"payload_json"`
				} `json:"events"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			type payload struct {
				Data struct {
					Name        string `json:"name"`
					PrevState   string `json:"prev_state"`
					NewState    string `json:"new_state"`
					InitiatedBy string `json:"initiated_by"`
				} `json:"data"`
			}

			headers := []string{"ID", "OUTLET", "FROM", "TO", "BY", "TIME"}
			rows := make([][]string, 0, len(resp.Events))
			for _, e := range resp.Events {
				var p payload
				_ = json.Unmarshal([]byte(e.PayloadJSON), &p)
				from := p.Data.PrevState
				if from == "" {
					from = "-"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", e.ID),
					p.Data.Name,
					from,
					p.Data.NewState,
					p.Data.InitiatedBy,
					formatTimestamp(e.TS),
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
