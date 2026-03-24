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
	cmd.AddCommand(newSystemBackupCmd(client))
	cmd.AddCommand(newSystemCleanupCmd(client))
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
					Firmware string `json:"firmware"`
					Hardware string `json:"hardware"`
				} `json:"controller"`
				Poller struct {
					LastPollTS          string `json:"last_poll_ts"`
					PollOK              bool   `json:"poll_ok"`
					PollIntervalSeconds int    `json:"poll_interval_seconds"`
				} `json:"poller"`
				DB struct {
					DuckDBSize int64 `json:"duckdb_size_bytes"`
					SQLiteSize int64 `json:"sqlite_size_bytes"`
				} `json:"db"`
			}
			if err := client.Get(cmd.Context(), "/api/system", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			pollStatus := ColorStatus("OK")
			if !resp.Poller.PollOK {
				pollStatus = colorRed + "STALE" + colorReset
			}

			lastPoll := formatTimestamp(resp.Poller.LastPollTS)
			if resp.Poller.LastPollTS != "" {
				if t, err := time.Parse(time.RFC3339, resp.Poller.LastPollTS); err == nil {
					lastPoll = formatRelativeTime(t)
				}
			}

			interval := fmt.Sprintf("%ds", resp.Poller.PollIntervalSeconds)

			fmt.Println()
			PrintSection("Controller", []KV{
				{"Serial", resp.Controller.Serial},
				{"Firmware", resp.Controller.Firmware},
				{"Hardware", resp.Controller.Hardware},
			})
			fmt.Println()
			PrintSection("Poller", []KV{
				{"Last poll", lastPoll},
				{"Status", pollStatus},
				{"Interval", interval},
			})
			fmt.Println()
			PrintSection("Database", []KV{
				{"DuckDB", formatBytes(resp.DB.DuckDBSize)},
				{"SQLite", formatBytes(resp.DB.SQLiteSize)},
			})
			fmt.Println()

			return nil
		},
	}
}

func newSystemBackupCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Trigger a database backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Status    string   `json:"status"`
				Paths     []string `json:"paths"`
				SizeBytes int64    `json:"size_bytes"`
				Error     string   `json:"error,omitempty"`
			}
			if err := client.Post(cmd.Context(), "/api/system/backup", nil, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Backup %s\n", ColorStatus(resp.Status))
			for _, p := range resp.Paths {
				fmt.Printf("  %s\n", p)
			}
			fmt.Printf("  Total size: %s\n", formatBytes(resp.SizeBytes))
			return nil
		},
	}
}

func newSystemCleanupCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Run data retention cleanup",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Deleted struct {
					ProbeReadings  int64 `json:"probe_readings"`
					OutletStates   int64 `json:"outlet_states"`
					PowerEvents    int64 `json:"power_events"`
					ControllerMeta int64 `json:"controller_meta"`
				} `json:"deleted"`
				RetentionDays int `json:"retention_days"`
			}
			if err := client.Post(cmd.Context(), "/api/system/cleanup", nil, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Cleanup complete (retention: %d days)\n", resp.RetentionDays)
			fmt.Printf("  Probe readings:  %d deleted\n", resp.Deleted.ProbeReadings)
			fmt.Printf("  Outlet states:   %d deleted\n", resp.Deleted.OutletStates)
			fmt.Printf("  Power events:    %d deleted\n", resp.Deleted.PowerEvents)
			fmt.Printf("  Controller meta: %d deleted\n", resp.Deleted.ControllerMeta)
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
