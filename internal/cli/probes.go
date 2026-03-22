package cli

import (
	"fmt"
	"math"
	"time"

	"github.com/spf13/cobra"
)

// NewProbesCmd returns the "probes" command group.
func NewProbesCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "probes",
		Short: "View probe readings and history",
	}
	cmd.AddCommand(newProbesCurrentCmd(client))
	cmd.AddCommand(newProbesHistoryCmd(client))
	return cmd
}

func newProbesCurrentCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current probe readings",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Probes []struct {
					Name        string  `json:"name"`
					DisplayName string  `json:"display_name"`
					Type        string  `json:"type"`
					Value       float64 `json:"value"`
					Unit        string  `json:"unit"`
					TS          string  `json:"ts"`
					Status      string  `json:"status"`
				} `json:"probes"`
				PolledAt string `json:"polled_at"`
			}
			if err := client.Get(cmd.Context(), "/api/probes", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			headers := []string{"PROBE", "VALUE", "UNIT", "STATUS", "UPDATED"}
			rows := make([][]string, 0, len(resp.Probes))
			for _, p := range resp.Probes {
				rows = append(rows, []string{
					p.DisplayName,
					fmt.Sprintf("%.2f", p.Value),
					p.Unit,
					ColorStatus(p.Status),
					formatTimestamp(p.TS),
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newProbesHistoryCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <name>",
		Short: "Show probe value history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			interval, _ := cmd.Flags().GetString("interval")

			path := "/api/probes/" + name + "/history?"
			if from != "" {
				path += "from=" + from + "&"
			}
			if to != "" {
				path += "to=" + to + "&"
			}
			if interval != "" {
				path += "interval=" + interval + "&"
			}

			var resp struct {
				Probe    string `json:"probe"`
				From     string `json:"from"`
				To       string `json:"to"`
				Interval string `json:"interval"`
				Data     []struct {
					TS    string  `json:"ts"`
					Value float64 `json:"value"`
				} `json:"data"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Data) == 0 {
				fmt.Println("No data points in range.")
				return nil
			}

			// Compute stats.
			var sum, min, max float64
			min = math.MaxFloat64
			max = -math.MaxFloat64
			for _, dp := range resp.Data {
				sum += dp.Value
				if dp.Value < min {
					min = dp.Value
				}
				if dp.Value > max {
					max = dp.Value
				}
			}
			avg := sum / float64(len(resp.Data))

			headers := []string{"TIMESTAMP", "VALUE"}
			rows := make([][]string, 0, len(resp.Data))
			for _, dp := range resp.Data {
				rows = append(rows, []string{
					formatTimestamp(dp.TS),
					fmt.Sprintf("%.2f", dp.Value),
				})
			}
			PrintTable(headers, rows)

			fmt.Printf("\n%d data points | min: %.2f | max: %.2f | avg: %.2f\n",
				len(resp.Data), min, max, avg)
			return nil
		},
	}
	cmd.Flags().String("from", "", "start time (RFC3339)")
	cmd.Flags().String("to", "", "end time (RFC3339)")
	cmd.Flags().String("interval", "", "bucket interval (e.g. 1m, 5m, 1h)")
	return cmd
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
