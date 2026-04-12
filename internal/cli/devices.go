package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewDevicesCmd returns the "devices" command group.
func NewDevicesCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices",
		Short: "View aquarium devices",
	}
	cmd.AddCommand(newDevicesListCmd(client))
	return cmd
}

func newDevicesListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Devices []struct {
					ID         int64   `json:"id"`
					Name       string  `json:"name"`
					DeviceType string  `json:"device_type"`
					Brand      *string `json:"brand"`
					Model      *string `json:"model"`
					OutletID   *string `json:"outlet_id"`
					Description *string `json:"description"`
				} `json:"devices"`
			}
			if err := client.Get(cmd.Context(), "/api/devices", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Devices) == 0 {
				fmt.Println("No devices configured.")
				return nil
			}

			headers := []string{"ID", "NAME", "TYPE", "BRAND/MODEL", "OUTLET"}
			rows := make([][]string, 0, len(resp.Devices))
			for _, d := range resp.Devices {
				brandModel := "-"
				if d.Brand != nil && d.Model != nil {
					brandModel = *d.Brand + " " + *d.Model
				} else if d.Brand != nil {
					brandModel = *d.Brand
				} else if d.Model != nil {
					brandModel = *d.Model
				}
				outlet := "-"
				if d.OutletID != nil {
					outlet = *d.OutletID
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", d.ID),
					d.Name,
					d.DeviceType,
					brandModel,
					outlet,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}
