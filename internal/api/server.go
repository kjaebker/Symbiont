package api

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
)

// Server is the HTTP API server.
type Server struct {
	duck        *db.DuckDB
	sqlite      *db.SQLiteDB
	apex        apex.Client
	cfg         *config.Config
	logger      *slog.Logger
	http        *http.Server
	broadcaster *Broadcaster
}

// New creates a new API server.
func New(cfg *config.Config, duck *db.DuckDB, sqlite *db.SQLiteDB, apexClient apex.Client, logger *slog.Logger) *Server {
	s := &Server{
		duck:        duck,
		sqlite:      sqlite,
		apex:        apexClient,
		cfg:         cfg,
		logger:      logger,
		broadcaster: NewBroadcaster(),
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Build middleware chain: RequestID → Logger → Recover → CORS → Auth → handler
	var handler http.Handler = mux
	handler = Auth(sqlite)(handler)
	handler = CORS(handler)
	handler = Recover(logger)(handler)
	handler = Logger(logger)(handler)
	handler = RequestID(handler)

	s.http = &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health check.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Probes.
	mux.HandleFunc("GET /api/probes", s.HandleProbeList)
	mux.HandleFunc("GET /api/probes/{name}/history", s.HandleProbeHistory)

	// Outlets.
	mux.HandleFunc("GET /api/outlets", s.HandleOutletList)
	mux.HandleFunc("PUT /api/outlets/{id}", s.HandleOutletSet)
	mux.HandleFunc("GET /api/outlets/events", s.HandleOutletEvents)

	// System.
	mux.HandleFunc("GET /api/system", s.HandleSystemStatus)

	// Config.
	mux.HandleFunc("GET /api/config/probes", s.HandleProbeConfigList)
	mux.HandleFunc("PUT /api/config/probes/{name}", s.HandleProbeConfigUpdate)
	mux.HandleFunc("GET /api/config/outlets", s.HandleOutletConfigList)
	mux.HandleFunc("PUT /api/config/outlets/{id}", s.HandleOutletConfigUpdate)

	// Alerts.
	mux.HandleFunc("GET /api/alerts", s.HandleAlertList)
	mux.HandleFunc("POST /api/alerts", s.HandleAlertCreate)
	mux.HandleFunc("PUT /api/alerts/{id}", s.HandleAlertUpdate)
	mux.HandleFunc("DELETE /api/alerts/{id}", s.HandleAlertDelete)

	// SSE stream.
	mux.HandleFunc("GET /api/stream", s.HandleStream)

	// Auth tokens.
	mux.HandleFunc("GET /api/tokens", s.HandleTokenList)
	mux.HandleFunc("POST /api/tokens", s.HandleTokenCreate)
	mux.HandleFunc("DELETE /api/tokens/{id}", s.HandleTokenDelete)

	// Static frontend serving with SPA fallback.
	mux.Handle("GET /", spaHandler(s.cfg.FrontendPath))
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	// Start background SSE poller.
	s.StartSSEPoller(ctx)

	// Start server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("api server starting", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server error: %w", err)
		}
		close(errCh)
	}()

	// Wait for context cancellation or server error.
	select {
	case <-ctx.Done():
		s.logger.Info("api server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// Addr returns the server's listener address. Only valid after Run has started.
func (s *Server) Addr() net.Addr {
	return nil // Will be useful for tests later if needed.
}

// spaHandler serves static files from dir. If the requested file doesn't exist,
// it falls back to index.html to support client-side routing (React Router).
func spaHandler(dir string) http.Handler {
	fsys := os.DirFS(dir)
	fileServer := http.FileServerFS(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file.
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		// Check if the file exists.
		if _, err := fs.Stat(fsys, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for any unmatched route.
		if _, err := fs.Stat(fsys, "index.html"); err != nil {
			http.NotFound(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
