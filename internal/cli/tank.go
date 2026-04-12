package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTankCmd returns the "tank" command group.
func NewTankCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tank",
		Short: "View and manage tank profile",
	}
	cmd.AddCommand(newTankProfileCmd(client))
	cmd.AddCommand(newTankSetCmd(client, "display"))
	cmd.AddCommand(newTankSetCmd(client, "sump"))
	return cmd
}

func newTankProfileCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "Show tank and sump profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Display *tankProfileJSON `json:"display"`
				Sump    *tankProfileJSON `json:"sump"`
			}
			if err := client.Get(cmd.Context(), "/api/tank/profile", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			printTankProfile("Display Tank", resp.Display)
			fmt.Println()
			printTankProfile("Sump", resp.Sump)
			return nil
		},
	}
}

type tankProfileJSON struct {
	Section       string   `json:"section"`
	DisplayName   *string  `json:"display_name"`
	VolumeGallons *float64 `json:"volume_gallons"`
	LengthIn      *float64 `json:"length_in"`
	WidthIn       *float64 `json:"width_in"`
	HeightIn      *float64 `json:"height_in"`
	TankType      *string  `json:"tank_type"`
	Manufacturer  *string  `json:"manufacturer"`
	Model         *string  `json:"model"`
	SetupDate     *string  `json:"setup_date"`
	Notes         *string  `json:"notes"`
}

func printTankProfile(label string, p *tankProfileJSON) {
	fmt.Printf("=== %s ===\n", label)
	if p == nil {
		fmt.Println("  (not configured)")
		return
	}
	pstr := func(v *string) string {
		if v == nil {
			return "-"
		}
		return *v
	}
	pfloat := func(v *float64) string {
		if v == nil {
			return "-"
		}
		return fmt.Sprintf("%.1f", *v)
	}
	if p.DisplayName != nil {
		fmt.Printf("  Name:         %s\n", *p.DisplayName)
	}
	fmt.Printf("  Type:         %s\n", pstr(p.TankType))
	fmt.Printf("  Volume:       %s gal\n", pfloat(p.VolumeGallons))
	if p.LengthIn != nil || p.WidthIn != nil || p.HeightIn != nil {
		fmt.Printf("  Dimensions:   %s\" × %s\" × %s\" (L×W×H)\n",
			pfloat(p.LengthIn), pfloat(p.WidthIn), pfloat(p.HeightIn))
	}
	fmt.Printf("  Manufacturer: %s\n", pstr(p.Manufacturer))
	fmt.Printf("  Model:        %s\n", pstr(p.Model))
	fmt.Printf("  Setup date:   %s\n", pstr(p.SetupDate))
	if p.Notes != nil {
		fmt.Printf("  Notes:        %s\n", *p.Notes)
	}
}

func newTankSetCmd(client *APIClient, section string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   section,
		Short: fmt.Sprintf("Update %s profile", section),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}

			setStr := func(flag, key string) {
				if cmd.Flags().Changed(flag) {
					v, _ := cmd.Flags().GetString(flag)
					body[key] = v
				}
			}
			setFloat := func(flag, key string) {
				if cmd.Flags().Changed(flag) {
					v, _ := cmd.Flags().GetFloat64(flag)
					body[key] = v
				}
			}

			setStr("name", "display_name")
			setStr("type", "tank_type")
			setFloat("gallons", "volume_gallons")
			setFloat("length", "length_in")
			setFloat("width", "width_in")
			setFloat("height", "height_in")
			setStr("manufacturer", "manufacturer")
			setStr("model", "model")
			setStr("setup-date", "setup_date")
			setStr("notes", "notes")

			if len(body) == 0 {
				return fmt.Errorf("provide at least one flag to update")
			}

			var resp any
			if err := client.Put(cmd.Context(), "/api/tank/profile/"+section, body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("%s profile updated\n", section)
			return nil
		},
	}
	cmd.Flags().String("name", "", "display name")
	cmd.Flags().String("type", "", "tank type: reef, fowlr, mixed, nano, freshwater, other")
	cmd.Flags().Float64("gallons", 0, "volume in gallons")
	cmd.Flags().Float64("length", 0, "length in inches")
	cmd.Flags().Float64("width", 0, "width in inches")
	cmd.Flags().Float64("height", 0, "height in inches")
	cmd.Flags().String("manufacturer", "", "tank manufacturer")
	cmd.Flags().String("model", "", "tank model")
	cmd.Flags().String("setup-date", "", "setup date (YYYY-MM-DD)")
	cmd.Flags().String("notes", "", "notes")
	return cmd
}
