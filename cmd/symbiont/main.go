package main

import (
	"fmt"
	"os"

	"github.com/kjaebker/symbiont/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "symbiont",
		Short: "Symbiont — Neptune Apex aquarium controller CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// auth reset doesn't need a token or API connection.
			if cmd.Name() == "reset" {
				return nil
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags.
	rootCmd.PersistentFlags().Bool("json", false, "output raw JSON")
	rootCmd.PersistentFlags().String("api-url", "http://localhost:8420", "API base URL")
	rootCmd.PersistentFlags().String("token", "", "API auth token")

	// Create API client lazily via a wrapper that resolves the token.
	var client *cli.APIClient
	getClient := func(cmd *cobra.Command) (*cli.APIClient, error) {
		if client != nil {
			return client, nil
		}
		token, err := cli.LoadToken(cmd.Root().PersistentFlags())
		if err != nil {
			return nil, err
		}
		apiURL, _ := cmd.Root().PersistentFlags().GetString("api-url")
		client = cli.NewAPIClient(apiURL, token)
		return client, nil
	}

	// We need a client for subcommands. Create a placeholder that will be
	// initialized in PersistentPreRunE. But cobra subcommands are registered
	// before the client exists, so we use a shared pointer.
	//
	// The approach: create a client with empty values, then fill them in
	// PersistentPreRunE before any RunE executes.
	sharedClient := &cli.APIClient{}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// auth reset doesn't need a token or API connection.
		if cmd.Name() == "reset" {
			return nil
		}
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		*sharedClient = *c
		return nil
	}

	// Register command groups.
	rootCmd.AddCommand(cli.NewProbesCmd(sharedClient))
	rootCmd.AddCommand(cli.NewOutletsCmd(sharedClient))
	rootCmd.AddCommand(cli.NewAlertsCmd(sharedClient))
	rootCmd.AddCommand(cli.NewSystemCmd(sharedClient))
	rootCmd.AddCommand(cli.NewAuthCmd(sharedClient))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
