package main

import (
	"fmt"
	"os"

	"github.com/kjaebker/symbiont/internal/cli"
	mcptools "github.com/kjaebker/symbiont/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	apiURL := envOrDefault("SYMBIONT_API_URL", "http://localhost:8420")
	token := os.Getenv("SYMBIONT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "SYMBIONT_TOKEN environment variable is required")
		os.Exit(1)
	}

	client := cli.NewAPIClient(apiURL, token)

	s := server.NewMCPServer(
		"symbiont",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	mcptools.RegisterTools(s, client)

	fmt.Fprintln(os.Stderr, "symbiont-mcp: starting stdio server")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "symbiont-mcp: server error: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
