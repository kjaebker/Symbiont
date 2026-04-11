package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewMeasurementsCmd returns the "measurements" command group.
func NewMeasurementsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "measurements",
		Aliases: []string{"meas"},
		Short:   "Manage manual water chemistry measurements",
	}
	cmd.AddCommand(newMeasurementsListCmd(client))
	cmd.AddCommand(newMeasurementsAddCmd(client))
	cmd.AddCommand(newMeasurementsDeleteCmd(client))
	cmd.AddCommand(newMeasurementsParametersCmd(client))
	return cmd
}

func newMeasurementsParametersCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "parameters",
		Short: "List available measurement parameters",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Parameters []struct {
					ID            int64  `json:"id"`
					Name          string `json:"name"`
					CanonicalUnit string `json:"canonical_unit"`
				} `json:"parameters"`
			}
			if err := client.Get(cmd.Context(), "/api/measurements/parameters", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Parameters) == 0 {
				fmt.Println("No parameters defined.")
				return nil
			}

			headers := []string{"ID", "PARAMETER", "UNIT"}
			rows := make([][]string, 0, len(resp.Parameters))
			for _, p := range resp.Parameters {
				rows = append(rows, []string{
					fmt.Sprintf("%d", p.ID),
					p.Name,
					p.CanonicalUnit,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newMeasurementsListCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List measurements",
		RunE: func(cmd *cobra.Command, args []string) error {
			parameter, _ := cmd.Flags().GetString("parameter")
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/measurements?"
			if parameter != "" {
				path += "parameter=" + parameter + "&"
			}
			if from != "" {
				path += "from=" + from + "&"
			}
			if to != "" {
				path += "to=" + to + "&"
			}
			if limit > 0 {
				path += fmt.Sprintf("limit=%d&", limit)
			}

			var resp struct {
				Measurements []struct {
					ID            int64   `json:"id"`
					MeasuredAt    string  `json:"measured_at"`
					Parameter     string  `json:"parameter"`
					CanonicalUnit string  `json:"canonical_unit"`
					Value         float64 `json:"value"`
					Notes         *string `json:"notes"`
					Source        string  `json:"source"`
					TestKitRef    *string `json:"test_kit_ref"`
				} `json:"measurements"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Measurements) == 0 {
				fmt.Println("No measurements found.")
				return nil
			}

			headers := []string{"ID", "DATE", "PARAMETER", "VALUE", "NOTES"}
			rows := make([][]string, 0, len(resp.Measurements))
			for _, m := range resp.Measurements {
				unit := m.CanonicalUnit
				valueStr := fmt.Sprintf("%.4g", m.Value)
				if unit != "" {
					valueStr += " " + unit
				}
				notes := "-"
				if m.Notes != nil && *m.Notes != "" {
					notes = *m.Notes
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", m.ID),
					formatTimestamp(m.MeasuredAt),
					m.Parameter,
					valueStr,
					notes,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().String("parameter", "", "filter by parameter name (e.g. Magnesium)")
	cmd.Flags().String("from", "", "start time (RFC3339)")
	cmd.Flags().String("to", "", "end time (RFC3339)")
	cmd.Flags().Int("limit", 0, "max results (default 200)")
	return cmd
}

func newMeasurementsAddCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new measurement",
		RunE: func(cmd *cobra.Command, args []string) error {
			parameter, _ := cmd.Flags().GetString("parameter")
			value, _ := cmd.Flags().GetFloat64("value")
			at, _ := cmd.Flags().GetString("at")
			notes, _ := cmd.Flags().GetString("notes")
			kitRef, _ := cmd.Flags().GetString("kit")

			measuredAt := time.Now().UTC().Format(time.RFC3339)
			if at != "" {
				t, err := time.Parse(time.RFC3339, at)
				if err != nil {
					return fmt.Errorf("invalid --at value (RFC3339 required): %w", err)
				}
				measuredAt = t.UTC().Format(time.RFC3339)
			}

			body := map[string]any{
				"parameter":   parameter,
				"value":       value,
				"measured_at": measuredAt,
			}
			if notes != "" {
				body["notes"] = notes
			}
			if kitRef != "" {
				body["test_kit_ref"] = kitRef
			}

			var resp struct {
				ID            int64   `json:"id"`
				MeasuredAt    string  `json:"measured_at"`
				Parameter     string  `json:"parameter"`
				CanonicalUnit string  `json:"canonical_unit"`
				Value         float64 `json:"value"`
			}
			if err := client.Post(cmd.Context(), "/api/measurements", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			unit := resp.CanonicalUnit
			valueStr := fmt.Sprintf("%.4g", resp.Value)
			if unit != "" {
				valueStr += " " + unit
			}
			fmt.Printf("Measurement #%d recorded: %s %s at %s\n",
				resp.ID, resp.Parameter, valueStr, formatTimestamp(resp.MeasuredAt))
			return nil
		},
	}
	cmd.Flags().String("parameter", "", "parameter name (required, e.g. Magnesium)")
	cmd.Flags().Float64("value", 0, "measured value (required)")
	cmd.Flags().String("at", "", "when the measurement was taken (RFC3339, default: now)")
	cmd.Flags().String("notes", "", "optional notes")
	cmd.Flags().String("kit", "", "test kit ref (e.g. hanna/hi772)")
	_ = cmd.MarkFlagRequired("parameter")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func newMeasurementsDeleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a measurement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Delete measurement #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/measurements/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "deleted", "id": id})
				return nil
			}

			fmt.Printf("Measurement #%s deleted\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}
