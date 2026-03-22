package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewSystemCmd returns the "system" command group.
func NewSystemCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "View system status",
	}
	cmd.AddCommand(newSystemStatusCmd(client))
	return cmd
}

func newSystemStatusCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Controller struct {
					Serial   string `json:"serial"`
					Hostname string `json:"hostname"`
					Software string `json:"software"`
					Hardware string `json:"hardware"`
					Type     string `json:"type"`
				} `json:"controller"`
				PollOK       bool   `json:"poll_ok"`
				LastPoll     string `json:"last_poll"`
				PollInterval string `json:"poll_interval"`
				DuckDBSize   int64  `json:"duckdb_size_bytes"`
				SQLiteSize   int64  `json:"sqlite_size_bytes"`
			}
			if err := client.Get(cmd.Context(), "/api/system", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			pollStatus := ColorStatus("OK")
			if !resp.PollOK {
				pollStatus = colorRed + "STALE" + colorReset
			}

			lastPoll := formatTimestamp(resp.LastPoll)
			if resp.LastPoll != "" {
				if t, err := time.Parse(time.RFC3339, resp.LastPoll); err == nil {
					lastPoll = formatRelativeTime(t)
				}
			}

			fmt.Println()
			PrintSection("Controller", []KV{
				{"Serial", resp.Controller.Serial},
				{"Hostname", resp.Controller.Hostname},
				{"Firmware", resp.Controller.Software},
				{"Hardware", resp.Controller.Hardware},
				{"Type", resp.Controller.Type},
			})
			fmt.Println()
			PrintSection("Poller", []KV{
				{"Last poll", lastPoll},
				{"Status", pollStatus},
				{"Interval", resp.PollInterval},
			})
			fmt.Println()
			PrintSection("Database", []KV{
				{"DuckDB", formatBytes(resp.DuckDBSize)},
				{"SQLite", formatBytes(resp.SQLiteSize)},
			})
			fmt.Println()

			return nil
		},
	}
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}
