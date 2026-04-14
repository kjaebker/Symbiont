package api

import (
	"fmt"
	"net/http"

	"github.com/kjaebker/symbiont/internal/cli"
	mcptools "github.com/kjaebker/symbiont/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newMCPHTTPHandler builds a Streamable-HTTP MCP handler that the API server
// mounts at /api/mcp.  Auth is already enforced by the Auth middleware before
// this handler runs.  Tool calls loop back to the API server via a local
// APIClient so they share the same validation layer as every other caller.
//
// token must be a valid admin-scoped token stored in SQLite.  If empty the
// handler is not mounted (HTTP MCP transport disabled).
func newMCPHTTPHandler(apiPort, token string) http.Handler {
	apiBase := fmt.Sprintf("http://localhost:%s", apiPort)
	client := cli.NewAPIClient(apiBase, token)

	mcpServer := server.NewMCPServer(
		"symbiont",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	mcptools.RegisterTools(mcpServer, client)

	return server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
	)
}
