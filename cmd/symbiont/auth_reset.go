package main

import (
	"context"
	"fmt"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/spf13/cobra"
)

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

			if cli.IsJSON(cmd) {
				cli.PrintJSON(map[string]string{"token": token})
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
