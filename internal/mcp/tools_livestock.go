package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- get_livestock ---

func getLivestockTool() mcp.Tool {
	return mcp.NewTool("get_livestock",
		mcp.WithDescription("Get the list of aquarium livestock — fish, coral, and invertebrates. Returns name, species, type, quantity, current health status, and date added. Use to understand what lives in the tank and the health of each resident."),
		mcp.WithString("type",
			mcp.Description("Filter by type: fish, coral, invertebrate, other. Omit for all."),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by health status: healthy, sick, quarantine, deceased. Omit for all."),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
	)
}

func getLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := request.GetString("type", ""); v != "" {
			q.Set("type", v)
		}
		if v := request.GetString("status", ""); v != "" {
			q.Set("status", v)
		}
		path := "/api/livestock?" + q.Encode()

		var resp any
		if err := apiCall(clientFromCtx(ctx, client), path, &resp); err != nil {
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_livestock ---

func addLivestockTool() mcp.Tool {
	return mcp.NewTool("add_livestock",
		mcp.WithDescription("Add a new livestock item (fish, coral, or invertebrate) to the tank log."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Common name — e.g. 'Ocellaris Clownfish', 'Acropora millepora'"),
		),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("Type of livestock"),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("species",
			mcp.Description("Scientific or species name — e.g. 'Amphiprion ocellaris'"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Number of this item (default 1)"),
		),
		mcp.WithString("status",
			mcp.Description("Initial health status (default: healthy)"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("date_added",
			mcp.Description("Date added to the tank (YYYY-MM-DD). Defaults to today if omitted."),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes"),
		),
	)
}

func addLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError("Parameter 'name' is required"), nil
		}
		livestockType, err := request.RequireString("type")
		if err != nil {
			return toolError("Parameter 'type' is required"), nil
		}

		body := map[string]any{
			"name": name,
			"type": livestockType,
		}
		if v := request.GetString("species", ""); v != "" {
			body["species"] = v
		}
		if v := request.GetInt("quantity", 0); v > 0 {
			body["quantity"] = v
		}
		if v := request.GetString("status", ""); v != "" {
			body["status"] = v
		}
		if v := request.GetString("date_added", ""); v != "" {
			body["date_added"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/livestock", body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to add livestock: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- update_livestock ---

func updateLivestockTool() mcp.Tool {
	return mcp.NewTool("update_livestock",
		mcp.WithDescription("Update an existing livestock item. Commonly used to record status changes — e.g. marking a fish as sick or deceased. The id must be the numeric ID from get_livestock."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithString("name",
			mcp.Description("Common name"),
		),
		mcp.WithString("type",
			mcp.Description("Type: fish, coral, invertebrate, other"),
			mcp.Enum("fish", "coral", "invertebrate", "other"),
		),
		mcp.WithString("species",
			mcp.Description("Scientific or species name"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Quantity"),
		),
		mcp.WithString("status",
			mcp.Description("Health status"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("date_added",
			mcp.Description("Date added (YYYY-MM-DD)"),
		),
		mcp.WithString("notes",
			mcp.Description("Notes"),
		),
	)
}

func updateLivestockHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		// Fetch existing item to merge changes.
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
		if err := apiCall(clientFromCtx(ctx, client), "/api/livestock/"+idStr, &existing); err != nil {
			return toolError(fmt.Sprintf("Livestock item %s not found: %v", idStr, err)), nil
		}

		body := map[string]any{
			"name":     existing.Name,
			"type":     existing.Type,
			"quantity": existing.Quantity,

			"status": existing.Status,
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

		// Apply provided overrides.
		if v := request.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := request.GetString("type", ""); v != "" {
			body["type"] = v
		}
		if v := request.GetString("species", ""); v != "" {
			body["species"] = v
		}
		if v := request.GetInt("quantity", 0); v > 0 {
			body["quantity"] = v
		}
		if v := request.GetString("status", ""); v != "" {
			body["status"] = v
		}
		if v := request.GetString("date_added", ""); v != "" {
			body["date_added"] = v
		}
		if v := request.GetString("notes", ""); v != "" {
			body["notes"] = v
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Put(timeoutCtx, "/api/livestock/"+idStr, body, &resp); err != nil {
			return toolError(fmt.Sprintf("Failed to update livestock: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_livestock_observations ---

func getLivestockObservationsTool() mcp.Tool {
	return mcp.NewTool("get_livestock_observations",
		mcp.WithDescription("Get the health observation history for a specific livestock item — timestamped entries with health status and notes. Use the id from get_livestock. Useful for tracking how a fish or coral has been doing over time."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
	)
}

func getLivestockObservationsHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		var resp any
		if err := apiCall(clientFromCtx(ctx, client), "/api/livestock/"+idStr+"/observations", &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- add_livestock_observation ---

func addLivestockObservationTool() mcp.Tool {
	return mcp.NewTool("add_livestock_observation",
		mcp.WithDescription("Log a health observation for a livestock item. Provide a status change, a note, or both. Use the id from get_livestock."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithString("status",
			mcp.Description("Health status at time of observation"),
			mcp.Enum("healthy", "sick", "quarantine", "deceased"),
		),
		mcp.WithString("note",
			mcp.Description("Observation note — describe what you observed"),
		),
	)
}

func addLivestockObservationHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		status := request.GetString("status", "")
		note := request.GetString("note", "")
		if status == "" && note == "" {
			return toolError("At least one of 'status' or 'note' is required"), nil
		}

		body := map[string]any{}
		if status != "" {
			body["status"] = status
		}
		if note != "" {
			body["note"] = note
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp any
		if err := clientFromCtx(ctx, client).Post(timeoutCtx, "/api/livestock/"+idStr+"/observations", body, &resp); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Failed to add observation: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

// --- get_livestock_image ---

func getLivestockImageTool() mcp.Tool {
	return mcp.NewTool("get_livestock_image",
		mcp.WithDescription("Fetch the photo for a livestock item so you can visually inspect it. Returns the image as inline content. Use the id from get_livestock. Pass thumbnail=true (the default) for a fast preview, or thumbnail=false for the full-resolution image."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Livestock item ID from get_livestock"),
		),
		mcp.WithBoolean("thumbnail",
			mcp.Description("Return the thumbnail instead of the full image (default: true)"),
		),
	)
}

func getLivestockImageHandler(client *cli.APIClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idFloat := request.GetFloat("id", 0)
		if idFloat == 0 {
			return toolError("Parameter 'id' is required"), nil
		}
		idStr := fmt.Sprintf("%d", int64(idFloat))

		// Default thumbnail=true unless explicitly set to false.
		useThumbnail := true
		if v, ok := request.GetArguments()["thumbnail"]; ok {
			if b, ok := v.(bool); ok {
				useThumbnail = b
			}
		}

		// Fetch the livestock item to get its image_path.
		var item struct {
			Name      string  `json:"name"`
			ImagePath *string `json:"image_path"`
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := clientFromCtx(ctx, client).Get(timeoutCtx, "/api/livestock/"+idStr, &item); err != nil {
			if apiErr, ok := err.(*cli.APIError); ok && apiErr.Status == 404 {
				return toolError(fmt.Sprintf("Livestock item %s not found", idStr)), nil
			}
			return toolError(fmt.Sprintf("Cannot reach Symbiont API: %v", err)), nil
		}
		if item.ImagePath == nil {
			return mcp.NewToolResultText(fmt.Sprintf("Livestock item %s (%s) has no image.", idStr, item.Name)), nil
		}

		imagePath := *item.ImagePath
		if useThumbnail {
			imagePath = thumbPathMCP(imagePath)
		}

		data, contentType, err := clientFromCtx(ctx, client).GetBytes(timeoutCtx, "/"+imagePath)
		if err != nil {
			// Fall back to full image if thumbnail doesn't exist yet.
			if useThumbnail {
				data, contentType, err = clientFromCtx(ctx, client).GetBytes(timeoutCtx, "/"+*item.ImagePath)
			}
			if err != nil {
				return toolError(fmt.Sprintf("Failed to fetch image: %v", err)), nil
			}
		}

		if contentType == "" {
			contentType = "image/jpeg"
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Photo of %s (livestock ID %s):", item.Name, idStr)),
				mcp.NewImageContent(encoded, contentType),
			},
		}, nil
	}
}

// thumbPathMCP derives the thumbnail path from an image path, mirroring the
// naming convention used by the API server's image upload handlers.
func thumbPathMCP(imagePath string) string {
	for i := len(imagePath) - 1; i >= 0 && imagePath[i] != '/'; i-- {
		if imagePath[i] == '.' {
			return imagePath[:i] + "-thumb.jpg"
		}
	}
	return imagePath + "-thumb.jpg"
}
