package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewMaintenanceCmd returns the "maintenance" command group.
func NewMaintenanceCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "maintenance",
		Aliases: []string{"maint"},
		Short:   "Manage recurring maintenance tasks",
	}
	cmd.AddCommand(newMaintenanceListCmd(client))
	cmd.AddCommand(newMaintenanceCompleteCmd(client))
	cmd.AddCommand(newMaintenanceDueCmd(client))
	return cmd
}

func newMaintenanceListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all maintenance tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Tasks []struct {
					ID              int64   `json:"id"`
					Name            string  `json:"name"`
					Description     *string `json:"description"`
					Frequency       string  `json:"frequency"`
					IntervalDays    *float64 `json:"interval_days"`
					Enabled         bool    `json:"enabled"`
					LastCompletedAt *string `json:"last_completed_at"`
					NextDueAt       *string `json:"next_due_at"`
				} `json:"tasks"`
			}
			if err := client.Get(cmd.Context(), "/api/maintenance/tasks", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Tasks) == 0 {
				fmt.Println("No maintenance tasks.")
				return nil
			}

			headers := []string{"ID", "NAME", "FREQUENCY", "LAST DONE", "NEXT DUE", "ENABLED"}
			rows := make([][]string, 0, len(resp.Tasks))
			for _, t := range resp.Tasks {
				freq := t.Frequency
				if t.IntervalDays != nil && t.Frequency == "every_n_days" {
					freq = fmt.Sprintf("every_%.4gd", *t.IntervalDays)
				}
				last := "-"
				if t.LastCompletedAt != nil {
					last = formatTimestamp(*t.LastCompletedAt)
				}
				next := "-"
				if t.NextDueAt != nil {
					next = formatTimestamp(*t.NextDueAt)
				}
				enabled := "yes"
				if !t.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", t.ID),
					t.Name,
					freq,
					last,
					next,
					enabled,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newMaintenanceCompleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete <task-id>",
		Short: "Mark a maintenance task as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			at, _ := cmd.Flags().GetString("at")
			notes, _ := cmd.Flags().GetString("notes")

			body := map[string]any{}
			if at != "" {
				t, err := time.Parse(time.RFC3339, at)
				if err != nil {
					return fmt.Errorf("invalid --at value (RFC3339 required): %w", err)
				}
				body["completed_at"] = t.UTC().Format(time.RFC3339)
			}
			if notes != "" {
				body["notes"] = notes
			}

			var resp struct {
				ID          int64  `json:"id"`
				TaskID      int64  `json:"task_id"`
				CompletedAt string `json:"completed_at"`
			}
			path := "/api/maintenance/tasks/" + args[0] + "/complete"
			if err := client.Post(cmd.Context(), path, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Task #%s marked done (log #%d) at %s\n",
				args[0], resp.ID, formatTimestamp(resp.CompletedAt))
			return nil
		},
	}
	cmd.Flags().String("at", "", "when the task was completed (RFC3339, default: now)")
	cmd.Flags().String("notes", "", "optional notes (e.g. 'changed 15 gallons')")
	return cmd
}

func newMaintenanceDueCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "due",
		Short: "Show tasks currently due or overdue",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Items []struct {
					Kind      string  `json:"kind"`
					ID        int64   `json:"id"`
					Label     string  `json:"label"`
					Detail    string  `json:"detail"`
					IsOverdue bool    `json:"is_overdue"`
					NextDueAt *string `json:"next_due_at"`
				} `json:"items"`
			}
			if err := client.Get(cmd.Context(), "/api/tasks/due", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Items) == 0 {
				fmt.Println("Nothing due.")
				return nil
			}

			headers := []string{"KIND", "ID", "LABEL", "DETAIL", "STATUS"}
			rows := make([][]string, 0, len(resp.Items))
			for _, item := range resp.Items {
				status := "due"
				if item.IsOverdue {
					status = "OVERDUE"
				}
				rows = append(rows, []string{
					item.Kind,
					fmt.Sprintf("%d", item.ID),
					item.Label,
					item.Detail,
					status,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}
