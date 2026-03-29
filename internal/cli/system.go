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
	cmd.AddCommand(newSystemBackupsCmd(client))
	cmd.AddCommand(newSystemCleanupCmd(client))
	cmd.AddCommand(newSystemLogCmd(client))
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

func newSystemBackupsCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "backups",
		Short: "List backup history",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Backups []struct {
					ID        int64   `json:"id"`
					TS        string  `json:"ts"`
					Status    string  `json:"status"`
					Path      *string `json:"path"`
					SizeBytes *int64  `json:"size_bytes"`
					Error     *string `json:"error"`
				} `json:"backups"`
			}
			if err := client.Get(cmd.Context(), "/api/system/backups", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Backups) == 0 {
				fmt.Println("No backups recorded.")
				return nil
			}

			headers := []string{"ID", "TIME", "STATUS", "SIZE", "PATH"}
			rows := make([][]string, 0, len(resp.Backups))
			for _, b := range resp.Backups {
				size := "-"
				if b.SizeBytes != nil {
					size = formatBytes(*b.SizeBytes)
				}
				path := "-"
				if b.Path != nil {
					path = *b.Path
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", b.ID),
					formatTimestamp(b.TS),
					ColorStatus(b.Status),
					size,
					path,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newSystemLogCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent system log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetString("limit")
			service, _ := cmd.Flags().GetString("service")

			path := "/api/system/log?"
			if limit != "" {
				path += "limit=" + limit + "&"
			}
			if service != "" {
				path += "service=" + service + "&"
			}

			var resp struct {
				Lines []struct {
					TS      string `json:"ts"`
					Service string `json:"service"`
					Level   string `json:"level"`
					Msg     string `json:"msg"`
				} `json:"lines"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Lines) == 0 {
				fmt.Println("No log entries (journalctl unavailable or no entries).")
				return nil
			}

			for _, l := range resp.Lines {
				level := l.Level
				switch l.Level {
				case "ERROR":
					level = colorRed + l.Level + colorReset
				case "WARN":
					level = colorYellow + l.Level + colorReset
				case "DEBUG":
					level = colorBlue + l.Level + colorReset
				}
				fmt.Printf("%s  %-7s  %-7s  %s\n",
					formatTimestamp(l.TS), l.Service, level, l.Msg)
			}
			return nil
		},
	}
	cmd.Flags().String("limit", "", "max log lines to return (default 200, max 500)")
	cmd.Flags().String("service", "", "filter by service: api, poller")
	return cmd
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
