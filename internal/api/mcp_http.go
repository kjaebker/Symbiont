package api

import (
	"fmt"
	"net/http"

	"github.com/kjaebker/symbiont/internal/cli"
	mcptools "github.com/kjaebker/symbiont/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newMCPHTTPHandler builds a Streamable-HTTP MCP handler that the API server
// mounts at /api/mcp. Auth is already enforced by the Auth middleware before
// this handler runs.
//
// Each request gets its own loopback APIClient keyed to the caller's token so
// that scope enforcement (read/control/admin) applies to MCP tool calls just
// as it does to direct REST calls.
//
// fallbackToken is used to register tools at startup (e.g. for server metadata)
// but per-request calls use the caller's token extracted from the request.
func newMCPHTTPHandler(apiPort, fallbackToken string) http.Handler {
	apiBase := fmt.Sprintf("http://localhost:%s", apiPort)
	fallbackClient := cli.NewAPIClient(apiBase, fallbackToken)

	mcpServer := server.NewMCPServer(
		"symbiont",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	mcptools.RegisterTools(mcpServer, fallbackClient)

	streamHandler := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
	)

	// Wrap the streamable handler to inject a per-request loopback client.
	// The caller's token is extracted here (same logic as Auth middleware) and
	// stored in context so tool handlers use it for loopback API calls.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != "" {
			requestClient := cli.NewAPIClient(apiBase, token)
			r = r.WithContext(mcptools.WithLoopbackClient(r.Context(), requestClient))
		}
		streamHandler.ServeHTTP(w, r)
	})
}
