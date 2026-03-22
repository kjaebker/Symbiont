package cli

import (
	"context"
	"fmt"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/spf13/cobra"
)

// NewAuthCmd returns the "auth" command group.
func NewAuthCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(newAuthTokensCmd(client))
	cmd.AddCommand(newAuthResetCmd())
	return cmd
}

func newAuthTokensCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage API tokens",
	}
	cmd.AddCommand(newTokensListCmd(client))
	cmd.AddCommand(newTokensCreateCmd(client))
	cmd.AddCommand(newTokensRevokeCmd(client))
	return cmd
}

func newTokensListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Tokens []struct {
					ID        int64   `json:"id"`
					Label     string  `json:"label"`
					CreatedAt string  `json:"created_at"`
					LastUsed  *string `json:"last_used"`
				} `json:"tokens"`
			}
			if err := client.Get(cmd.Context(), "/api/tokens", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			headers := []string{"ID", "LABEL", "CREATED", "LAST USED"}
			rows := make([][]string, 0, len(resp.Tokens))
			for _, t := range resp.Tokens {
				lastUsed := "never"
				if t.LastUsed != nil {
					lastUsed = formatTimestamp(*t.LastUsed)
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", t.ID),
					t.Label,
					formatTimestamp(t.CreatedAt),
					lastUsed,
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

func newTokensCreateCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			label, _ := cmd.Flags().GetString("label")
			body := map[string]string{"label": label}

			var resp struct {
				ID    int64  `json:"id"`
				Label string `json:"label"`
				Token string `json:"token"`
			}
			if err := client.Post(cmd.Context(), "/api/tokens", body, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			fmt.Println("Token created (save this — shown once):")
			fmt.Println(resp.Token)

			// Offer to save to config file.
			fmt.Print("\nSave token to ~/.config/symbiont/token? [y/N] ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm == "y" || confirm == "Y" {
				if err := SaveToken(resp.Token); err != nil {
					return fmt.Errorf("saving token: %w", err)
				}
				fmt.Println("Token saved.")
			}

			return nil
		},
	}
	cmd.Flags().String("label", "", "token label (required)")
	_ = cmd.MarkFlagRequired("label")
	return cmd
}

func newTokensRevokeCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke an API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Revoke token #%s? [y/N] ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := client.Delete(cmd.Context(), "/api/tokens/"+id); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"status": "revoked", "id": id})
				return nil
			}

			fmt.Printf("Token #%s revoked\n", id)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}

func newAuthResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset all tokens (emergency recovery, requires --db-path)",
		Long:  "Directly accesses the SQLite database to delete all tokens and create a new default token. Use when you've lost your token and can't access the API.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, _ := cmd.Flags().GetString("db-path")
			yes, _ := cmd.Flags().GetBool("yes")

			if !yes {
				return fmt.Errorf("--yes flag is required for this destructive operation")
			}

			// Open SQLite directly (no API needed).
			sqliteDB, err := db.OpenSQLite(dbPath)
			if err != nil {
				return fmt.Errorf("opening sqlite at %s: %w", dbPath, err)
			}
			defer sqliteDB.Close()

			// Delete all tokens.
			if _, err := sqliteDB.DB().ExecContext(context.Background(), "DELETE FROM auth_tokens"); err != nil {
				return fmt.Errorf("deleting tokens: %w", err)
			}

			// Create new default token.
			token, _, err := sqliteDB.EnsureDefaultToken(context.Background())
			if err != nil {
				return fmt.Errorf("creating default token: %w", err)
			}

			if IsJSON(cmd) {
				PrintJSON(map[string]string{"token": token})
				return nil
			}

			fmt.Println("All tokens deleted. New default token:")
			fmt.Println(token)
			return nil
		},
	}
	cmd.Flags().String("db-path", "", "path to SQLite database (required)")
	cmd.Flags().Bool("yes", false, "confirm destructive operation")
	_ = cmd.MarkFlagRequired("db-path")
	return cmd
}
