package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

// NewLivestockCmd returns the "livestock" command group.
func NewLivestockCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "livestock",
		Short: "Manage tank livestock",
	}
	cmd.AddCommand(newLivestockListCmd(client))
	cmd.AddCommand(newLivestockAddCmd(client))
	cmd.AddCommand(newLivestockUpdateCmd(client))
	cmd.AddCommand(newLivestockDeleteCmd(client))
	cmd.AddCommand(newLivestockObserveCmd(client))
	cmd.AddCommand(newLivestockObservationsCmd(client))
	return cmd
}

func newLivestockListCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List livestock",
		RunE: func(cmd *cobra.Command, args []string) error {
			typeFilter, _ := cmd.Flags().GetString("type")
			statusFilter, _ := cmd.Flags().GetString("status")

			path := "/api/livestock"
			q := url.Values{}
			if typeFilter != "" {
				q.Set("type", typeFilter)
			}
			if statusFilter != "" {
				q.Set("status", statusFilter)
			}
			if len(q) > 0 {
				path += "?" + q.Encode()
			}

			var resp struct {
				Livestock []struct {
					ID        int64   `json:"id"`
					Name      string  `json:"name"`
					Species   *string `json:"species"`
					Type      string  `json:"type"`
					Quantity  int     `json:"quantity"`
					Status    string  `json:"status"`
					DateAdded *string `json:"date_added"`
				} `json:"livestock"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Livestock) == 0 {
				fmt.Println("No livestock.")
				return nil
			}

			headers := []string{"ID", "NAME", "SPECIES", "TYPE", "QTY", "STATUS", "ADDED"}
			rows := make([][]string, 0, len(resp.Livestock))
			for _, item := range resp.Livestock {
				species := "-"
				if item.Species != nil {
					species = *item.Species
				}
				added := "-"
				if item.DateAdded != nil {
					added = *item.DateAdded
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", item.ID),
					item.Name,
					species,
					item.Type,
					fmt.Sprintf("%d", item.Quantity),
					item.Status,
					added,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().String("type", "", "filter by type: fish, coral, invertebrate, other")
	cmd.Flags().String("status", "", "filter by status: healthy, sick, quarantine, deceased")
	return cmd
}

func newLivestockAddCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a livestock item",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			livestockType, _ := cmd.Flags().GetString("type")

			body := map[string]any{
				"name": name,
				"type": livestockType,
			}
			if cmd.Flags().Changed("species") {
				v, _ := cmd.Flags().GetString("species")
				body["species"] = v
			}
			if cmd.Flags().Changed("quantity") {
				v, _ := cmd.Flags().GetInt("quantity")
				body["quantity"] = v
			}
			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				body["status"] = v
			}
			if cmd.Flags().Changed("date-added") {
				v, _ := cmd.Flags().GetString("date-added")
				body["date_added"] = v
			}
			if cmd.Flags().Changed("notes") {
				v, _ := cmd.Flags().GetString("notes")
				body["notes"] = v
			}

			var resp struct {
				ID int64 `json:"id"`
			}
			if err := client.Post(cmd.Context(), "/api/livestock", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Livestock #%d added\n", resp.ID)
			return nil
		},
	}
	cmd.Flags().String("name", "", "common name (required)")
	cmd.Flags().String("type", "", "type: fish, coral, invertebrate, other (required)")
	cmd.Flags().String("species", "", "scientific/species name")
	cmd.Flags().Int("quantity", 1, "quantity")
	cmd.Flags().String("status", "healthy", "status: healthy, sick, quarantine, deceased")
	cmd.Flags().String("date-added", "", "date added (YYYY-MM-DD)")
	cmd.Flags().String("notes", "", "notes")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newLivestockUpdateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a livestock item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// Fetch existing item so we can do a full PUT with merged values.
			var existing struct {
				ID        int64   `json:"id"`
				Name      string  `json:"name"`
				Species   *string `json:"species"`
				Type      string  `json:"type"`
				Quantity  int     `json:"quantity"`
				Status    string  `json:"status"`
				DateAdded *string `json:"date_added"`
				Notes     *string `json:"notes"`
			}
			if err := client.Get(cmd.Context(), "/api/livestock/"+id, &existing); err != nil {
				return err
			}

			body := map[string]any{
				"name":     existing.Name,
				"type":     existing.Type,
				"quantity": existing.Quantity,
				"status":   existing.Status,
			}
			if existing.Species != nil {
				body["species"] = *existing.Species
			}
			if existing.DateAdded != nil {
				body["date_added"] = *existing.DateAdded
			}
			if existing.Notes != nil {
				body["notes"] = *existing.Notes
			}

			// Override with any flags that were explicitly set.
			if cmd.Flags().Changed("name") {
				v, _ := cmd.Flags().GetString("name")
				body["name"] = v
			}
			if cmd.Flags().Changed("type") {
				v, _ := cmd.Flags().GetString("type")
				body["type"] = v
			}
			if cmd.Flags().Changed("species") {
				v, _ := cmd.Flags().GetString("species")
				body["species"] = v
			}
			if cmd.Flags().Changed("quantity") {
				v, _ := cmd.Flags().GetInt("quantity")
				body["quantity"] = v
			}
			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				body["status"] = v
			}
			if cmd.Flags().Changed("date-added") {
				v, _ := cmd.Flags().GetString("date-added")
				body["date_added"] = v
			}
			if cmd.Flags().Changed("notes") {
				v, _ := cmd.Flags().GetString("notes")
				body["notes"] = v
			}

			if err := client.Put(cmd.Context(), "/api/livestock/"+id, body, nil); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "updated", "id": id})
				return nil
			}

			fmt.Printf("Livestock #%s updated\n", id)
			return nil
		},
	}
	cmd.Flags().String("name", "", "common name")
	cmd.Flags().String("type", "", "type: fish, coral, invertebrate, other")
	cmd.Flags().String("species", "", "scientific/species name")
	cmd.Flags().Int("quantity", 1, "quantity")
	cmd.Flags().String("status", "", "status: healthy, sick, quarantine, deceased")
	cmd.Flags().String("date-added", "", "date added (YYYY-MM-DD)")
	cmd.Flags().String("notes", "", "notes")
	return cmd
}

func newLivestockDeleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a livestock item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Delete livestock #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/livestock/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "deleted", "id": id})
				return nil
			}

			fmt.Printf("Livestock #%s deleted\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}

func newLivestockObserveCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observe <id>",
		Short: "Log an observation for a livestock item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			body := map[string]any{}
			if cmd.Flags().Changed("status") {
				v, _ := cmd.Flags().GetString("status")
				body["status"] = v
			}
			if cmd.Flags().Changed("note") {
				v, _ := cmd.Flags().GetString("note")
				body["note"] = v
			}
			if len(body) == 0 {
				return fmt.Errorf("at least one of --status or --note is required")
			}

			var resp struct {
				ID int64 `json:"id"`
			}
			if err := client.Post(cmd.Context(), "/api/livestock/"+id+"/observations", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Observation #%d logged for livestock #%s\n", resp.ID, id)
			return nil
		},
	}
	cmd.Flags().String("status", "", "health status: healthy, sick, quarantine, deceased")
	cmd.Flags().String("note", "", "observation note")
	return cmd
}

func newLivestockObservationsCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "observations <id>",
		Short: "List health observations for a livestock item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			var resp struct {
				Observations []struct {
					ID     int64   `json:"id"`
					TS     string  `json:"ts"`
					Status *string `json:"status"`
					Note   *string `json:"note"`
				} `json:"observations"`
			}
			if err := client.Get(cmd.Context(), "/api/livestock/"+id+"/observations", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Observations) == 0 {
				fmt.Printf("No observations for livestock #%s.\n", id)
				return nil
			}

			headers := []string{"ID", "TIMESTAMP", "STATUS", "NOTE"}
			rows := make([][]string, 0, len(resp.Observations))
			for _, o := range resp.Observations {
				status := "-"
				if o.Status != nil {
					status = *o.Status
				}
				note := "-"
				if o.Note != nil {
					note = *o.Note
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", o.ID),
					o.TS,
					status,
					note,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}
