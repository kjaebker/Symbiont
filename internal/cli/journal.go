package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

// NewJournalCmd returns the "journal" command group.
func NewJournalCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Manage the tank journal",
	}
	cmd.AddCommand(newJournalListCmd(client))
	cmd.AddCommand(newJournalAddCmd(client))
	cmd.AddCommand(newJournalDeleteCmd(client))
	return cmd
}

func newJournalListCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List journal entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			sentiment, _ := cmd.Flags().GetString("sentiment")
			limit, _ := cmd.Flags().GetInt("limit")

			path := "/api/journal"
			q := url.Values{}
			if category != "" {
				q.Set("category", category)
			}
			if sentiment != "" {
				q.Set("sentiment", sentiment)
			}
			if limit > 0 {
				q.Set("limit", fmt.Sprintf("%d", limit))
			}
			if len(q) > 0 {
				path += "?" + q.Encode()
			}

			var resp struct {
				Entries []struct {
					ID        int64   `json:"id"`
					TS        string  `json:"ts"`
					Category  string  `json:"category"`
					Sentiment *string `json:"sentiment"`
					Title     string  `json:"title"`
					Body      *string `json:"body"`
					Source    string  `json:"source"`
				} `json:"entries"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Entries) == 0 {
				fmt.Println("No journal entries.")
				return nil
			}

			headers := []string{"ID", "TIMESTAMP", "CATEGORY", "SENTIMENT", "TITLE", "SOURCE"}
			rows := make([][]string, 0, len(resp.Entries))
			for _, e := range resp.Entries {
				sentiment := "-"
				if e.Sentiment != nil {
					sentiment = *e.Sentiment
				}
				// Truncate timestamp to date+time without sub-seconds
				ts := e.TS
				if len(ts) > 19 {
					ts = ts[:19]
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", e.ID),
					ts,
					e.Category,
					sentiment,
					e.Title,
					e.Source,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().String("category", "", "filter by category: observation, maintenance, event, milestone")
	cmd.Flags().String("sentiment", "", "filter by sentiment: good, neutral, bad, critical")
	cmd.Flags().Int("limit", 50, "max entries to return (default 50)")
	return cmd
}

func newJournalAddCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a journal entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			title, _ := cmd.Flags().GetString("title")
			category, _ := cmd.Flags().GetString("category")

			body := map[string]any{
				"title":    title,
				"category": category,
			}
			if cmd.Flags().Changed("sentiment") {
				v, _ := cmd.Flags().GetString("sentiment")
				body["sentiment"] = v
			}
			if cmd.Flags().Changed("body") {
				v, _ := cmd.Flags().GetString("body")
				body["body"] = v
			}

			var resp struct {
				ID int64 `json:"id"`
			}
			if err := client.Post(cmd.Context(), "/api/journal", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Printf("Journal entry #%d added\n", resp.ID)
			return nil
		},
	}
	cmd.Flags().String("title", "", "entry title (required)")
	cmd.Flags().String("category", "observation", "category: observation, maintenance, event, milestone")
	cmd.Flags().String("sentiment", "", "sentiment: good, neutral, bad, critical")
	cmd.Flags().String("body", "", "additional details/notes")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newJournalDeleteCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a journal entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Delete journal entry #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/journal/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "deleted", "id": id})
				return nil
			}

			fmt.Printf("Journal entry #%s deleted\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}
