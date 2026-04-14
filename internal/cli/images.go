package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NewImagesCmd returns the top-level "images" command group.
func NewImagesCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Image utilities",
	}
	cmd.AddCommand(newImagesReprocessCmd(client))
	return cmd
}

// newImagesReprocessCmd calls the API to regenerate thumbnails for all stored
// livestock and observation images.
func newImagesReprocessCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reprocess",
		Short: "Regenerate thumbnails for all stored livestock images",
		Long: `Reads every livestock and observation image from disk and regenerates
its thumbnail (-thumb.jpg). Skips images that already have a thumbnail unless
--force is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			path := "/api/images/reprocess"
			if force {
				path += "?force=true"
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			var result struct {
				Total     int      `json:"total"`
				Processed int      `json:"processed"`
				Skipped   int      `json:"skipped"`
				Failed    int      `json:"failed"`
				Failures  []string `json:"failures"`
			}
			if err := client.Post(ctx, path, nil, &result); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(result)
				return nil
			}

			fmt.Printf("Total images: %d\n", result.Total)
			fmt.Printf("Processed:    %d\n", result.Processed)
			fmt.Printf("Skipped:      %d  (thumbnail already exists)\n", result.Skipped)
			fmt.Printf("Failed:       %d\n", result.Failed)
			for _, f := range result.Failures {
				fmt.Printf("  ! %s\n", f)
			}
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "regenerate thumbnails even if they already exist")
	return cmd
}

// newLivestockImageCmd shows image URLs for a livestock item and optionally
// downloads the image to a local file.
func newLivestockImageCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image <id>",
		Short: "Show or download the photo for a livestock item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			save, _ := cmd.Flags().GetString("save")
			thumbnail, _ := cmd.Flags().GetBool("thumbnail")

			// Fetch the livestock item to get image_path.
			var item struct {
				Name      string  `json:"name"`
				ImagePath *string `json:"image_path"`
			}
			if err := client.Get(cmd.Context(), "/api/livestock/"+id, &item); err != nil {
				return err
			}
			if item.ImagePath == nil {
				fmt.Printf("Livestock #%s (%s) has no image.\n", id, item.Name)
				return nil
			}

			relPath := *item.ImagePath
			thumbRelPath := derivedThumbPath(relPath)

			if IsJSON(cmd) {
				PrintJSON(map[string]string{
					"image_path":     relPath,
					"image_url":      client.BaseURL + "/" + relPath,
					"thumbnail_path": thumbRelPath,
					"thumbnail_url":  client.BaseURL + "/" + thumbRelPath,
				})
				return nil
			}

			fmt.Printf("Livestock:     #%s (%s)\n", id, item.Name)
			fmt.Printf("Full image:    %s/%s\n", client.BaseURL, relPath)
			fmt.Printf("Thumbnail:     %s/%s\n", client.BaseURL, thumbRelPath)

			if save != "" {
				fetchPath := relPath
				if thumbnail {
					fetchPath = thumbRelPath
				}
				if err := downloadImage(cmd.Context(), client, fetchPath, save); err != nil {
					return fmt.Errorf("downloading image: %w", err)
				}
				fmt.Printf("Saved to:      %s\n", save)
			}
			return nil
		},
	}
	cmd.Flags().String("save", "", "save the image to this local file path")
	cmd.Flags().Bool("thumbnail", false, "download the thumbnail instead of the full image (use with --save)")
	return cmd
}

// downloadImage fetches an image from /<relPath> and writes it to dest.
func downloadImage(ctx context.Context, client *APIClient, relPath, dest string) error {
	data, _, err := client.GetBytes(ctx, "/"+relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	return os.WriteFile(dest, data, 0o644)
}

// derivedThumbPath mirrors the server-side thumbPath() convention.
func derivedThumbPath(imagePath string) string {
	for i := len(imagePath) - 1; i >= 0 && imagePath[i] != '/'; i-- {
		if imagePath[i] == '.' {
			return imagePath[:i] + "-thumb.jpg"
		}
	}
	return imagePath + "-thumb.jpg"
}
