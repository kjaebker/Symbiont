package cli

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"
)

// NewDosingCmd returns the "dosing" command group.
func NewDosingCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dosing",
		Short: "Manage dosing schedules and logs",
	}
	cmd.AddCommand(newDosingListCmd(client))
	cmd.AddCommand(newDosingLogCmd(client))
	cmd.AddCommand(newDosingHistoryCmd(client))
	return cmd
}

func newDosingListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List dosing schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Schedules []struct {
					ID          int64   `json:"id"`
					ProductBrand string `json:"product_brand"`
					ProductName  string `json:"product_name"`
					ProductUnit  string `json:"product_unit"`
					Amount      float64 `json:"amount"`
					Frequency   string  `json:"frequency"`
					IntervalDays *float64 `json:"interval_days"`
					Enabled     bool    `json:"enabled"`
					LastCompletedAt *string `json:"last_completed_at"`
					NextDueAt   *string `json:"next_due_at"`
				} `json:"schedules"`
			}
			if err := client.Get(cmd.Context(), "/api/dosing/schedules", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Schedules) == 0 {
				fmt.Println("No dosing schedules.")
				return nil
			}

			headers := []string{"ID", "PRODUCT", "AMOUNT", "FREQUENCY", "LAST DOSED", "NEXT DUE", "ENABLED"}
			rows := make([][]string, 0, len(resp.Schedules))
			for _, s := range resp.Schedules {
				freq := s.Frequency
				if s.IntervalDays != nil && s.Frequency == "every_n_days" {
					freq = fmt.Sprintf("every_%.4gd", *s.IntervalDays)
				}
				last := "-"
				if s.LastCompletedAt != nil {
					last = formatTimestamp(*s.LastCompletedAt)
				}
				next := "-"
				if s.NextDueAt != nil {
					next = formatTimestamp(*s.NextDueAt)
				}
				enabled := "yes"
				if !s.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", s.ID),
					s.ProductBrand + " " + s.ProductName,
					fmt.Sprintf("%.4g %s", s.Amount, s.ProductUnit),
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

func newDosingLogCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log <schedule-id>",
		Short: "Record a dose against a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			at, _ := cmd.Flags().GetString("at")
			notes, _ := cmd.Flags().GetString("notes")
			amount, _ := cmd.Flags().GetFloat64("amount")

			body := map[string]any{}
			if at != "" {
				t, err := time.Parse(time.RFC3339, at)
				if err != nil {
					return fmt.Errorf("invalid --at value (RFC3339 required): %w", err)
				}
				body["dosed_at"] = t.UTC().Format(time.RFC3339)
			}
			if notes != "" {
				body["notes"] = notes
			}
			if amount > 0 {
				body["amount"] = amount
			}

			var resp struct {
				ID        int64   `json:"id"`
				Amount    float64 `json:"amount"`
				ProductBrand string `json:"product_brand"`
				ProductName  string `json:"product_name"`
				ProductUnit  string `json:"product_unit"`
				DosedAt   string  `json:"dosed_at"`
			}
			path := "/api/dosing/schedules/" + args[0] + "/log"
			if err := client.Post(cmd.Context(), path, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Dose #%d logged: %.4g %s %s %s at %s\n",
				resp.ID, resp.Amount, resp.ProductUnit,
				resp.ProductBrand, resp.ProductName,
				formatTimestamp(resp.DosedAt))
			return nil
		},
	}
	cmd.Flags().String("at", "", "when the dose was given (RFC3339, default: now)")
	cmd.Flags().String("notes", "", "optional notes")
	cmd.Flags().Float64("amount", 0, "override the scheduled amount")
	return cmd
}

func newDosingHistoryCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show recent dosing history",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")

			q := url.Values{}
			if limit > 0 {
				q.Set("limit", fmt.Sprintf("%d", limit))
			}
			path := "/api/dosing/logs"
			if len(q) > 0 {
				path += "?" + q.Encode()
			}

			var resp struct {
				Logs []struct {
					ID           int64   `json:"id"`
					ProductBrand string  `json:"product_brand"`
					ProductName  string  `json:"product_name"`
					ProductUnit  string  `json:"product_unit"`
					Amount       float64 `json:"amount"`
					DosedAt      string  `json:"dosed_at"`
					Notes        *string `json:"notes"`
					Source       string  `json:"source"`
				} `json:"logs"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Logs) == 0 {
				fmt.Println("No dosing history.")
				return nil
			}

			headers := []string{"ID", "DATE", "PRODUCT", "AMOUNT", "NOTES", "SOURCE"}
			rows := make([][]string, 0, len(resp.Logs))
			for _, l := range resp.Logs {
				notes := "-"
				if l.Notes != nil && *l.Notes != "" {
					notes = *l.Notes
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", l.ID),
					formatTimestamp(l.DosedAt),
					l.ProductBrand + " " + l.ProductName,
					fmt.Sprintf("%.4g %s", l.Amount, l.ProductUnit),
					notes,
					l.Source,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().Int("limit", 30, "max entries to return")
	return cmd
}
