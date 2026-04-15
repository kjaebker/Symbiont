package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kjaebker/symbiont/internal/db"
)

type contextKey string

const (
	requestIDKey  contextKey = "request_id"
	tokenScopeKey contextKey = "token_scope"
	tokenLabelKey contextKey = "token_label"
)

// TokenScopeFromContext returns the scope of the authenticated token, or
// empty string if not set (unauthenticated or pre-scope tokens).
func TokenScopeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tokenScopeKey).(string); ok {
		return v
	}
	return ""
}

// TokenLabelFromContext returns the label of the authenticated token.
func TokenLabelFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tokenLabelKey).(string); ok {
		return v
	}
	return ""
}

// RequestIDFromContext returns the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID generates a UUID and attaches it to the request context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wrote {
		rw.status = code
		rw.wrote = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wrote {
		rw.status = http.StatusOK
		rw.wrote = true
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher so SSE streaming works through the logger middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logger logs each request with structured fields.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

// Recover catches panics and returns a 500 error.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"err", fmt.Sprintf("%v", err),
						"stack", string(debug.Stack()),
						"request_id", RequestIDFromContext(r.Context()),
					)
					writeError(w, http.StatusInternalServerError, "internal server error", "internal_error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS allows requests from the Vite dev server and same-origin.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Auth validates the Bearer token via SQLite. Skips auth for the SSE stream
// endpoint (which uses a query param token instead) and for non-API paths
// (static frontend files).
//
// On success the token's scope and label are stored in the request context so
// handlers and downstream middleware can read them via TokenScopeFromContext
// and TokenLabelFromContext.
func Auth(sqlite *db.SQLiteDB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for SSE stream (uses ?token= query param) and health check.
			if r.URL.Path == "/api/stream" || r.URL.Path == "/api/health" || r.URL.Path == "/api/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for non-API paths (static frontend files).
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			token := extractBearerToken(r)
			// Fall back to ?token= query param for browser downloads (export, etc.)
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization token", "unauthorized")
				return
			}

			valid, ta := sqlite.ValidateToken(r.Context(), token)
			if !valid {
				writeError(w, http.StatusUnauthorized, "invalid authorization token", "unauthorized")
				return
			}

			// Enforce scope. Read-only tokens may not mutate state.
			// Exception: POST to /api/mcp is allowed because the MCP Streamable HTTP
			// transport uses POST for all operations regardless of whether the underlying
			// tool is read-only. Scope is enforced via the per-request loopback API client
			// in mcp_http.go, which uses the caller's token and will 403 on any actual
			// mutation attempt.
			if ta.Scope == "read" && r.Method != http.MethodGet && r.Method != http.MethodOptions {
				if r.URL.Path != "/api/mcp" {
					writeError(w, http.StatusForbidden, "token scope 'read' does not permit this operation", "forbidden")
					return
				}
			}
			// Control tokens may not manage tokens, config, backup, or agent settings.
			if ta.Scope == "control" && isAdminRoute(r) {
				writeError(w, http.StatusForbidden, "token scope 'control' does not permit this operation", "forbidden")
				return
			}

			// Store scope and label in context for downstream use.
			ctx := context.WithValue(r.Context(), tokenScopeKey, ta.Scope)
			ctx = context.WithValue(ctx, tokenLabelKey, ta.Label)

			// Update last_used asynchronously — don't block the request.
			go func() {
				_ = sqlite.TouchToken(context.Background(), ta.ID)
			}()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isAdminRoute returns true for routes that require admin scope.
func isAdminRoute(r *http.Request) bool {
	p := r.URL.Path
	m := r.Method
	// Token management.
	if strings.HasPrefix(p, "/api/tokens") {
		return true
	}
	// Probe and outlet config.
	if strings.HasPrefix(p, "/api/config/") && (m == http.MethodPut || m == http.MethodPost || m == http.MethodDelete) {
		return true
	}
	// Tank profile writes.
	if strings.HasPrefix(p, "/api/tank/profile") && (m == http.MethodPut || m == http.MethodPost) {
		return true
	}
	// Agent settings writes.
	if p == "/api/agent/settings" && m == http.MethodPut {
		return true
	}
	// System / backup.
	if strings.HasPrefix(p, "/api/system/backup") || strings.HasPrefix(p, "/api/system/cleanup") {
		return true
	}
	return false
}

// SecurityHeaders sets standard security headers on every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		// Permit inline styles (Tailwind generates them) and blob/data for images.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:")
		next.ServeHTTP(w, r)
	})
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
