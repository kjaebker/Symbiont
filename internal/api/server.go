package api

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kjaebker/symbiont/internal/apex"
	"github.com/kjaebker/symbiont/internal/config"
	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/events"
	"github.com/kjaebker/symbiont/internal/journal"
	"github.com/kjaebker/symbiont/internal/kits"
)


// Server is the HTTP API server.
type Server struct {
	duck             *db.DuckDB
	sqlite           *db.SQLiteDB
	apex             apex.Client
	cfg              *config.Config
	logger           *slog.Logger
	http             *http.Server
	broadcaster      *Broadcaster
	frontendFS       fs.FS
	catalog          *kits.Catalog
	events           *events.Bus
	journalTemplates *journal.Catalog
}

// New creates a new API server. frontendFS is the filesystem to serve the
// frontend from; if nil, falls back to os.DirFS(cfg.FrontendPath).
// catalog is the test kit catalog; if nil, kit-ref validation is skipped.
// bus is the system event bus; if nil, event publishing is a no-op.
// journalCatalog is the journal template catalog; if nil, templates return empty.
func New(cfg *config.Config, duck *db.DuckDB, sqlite *db.SQLiteDB, apexClient apex.Client, logger *slog.Logger, frontendFS fs.FS, catalog *kits.Catalog, bus *events.Bus, journalCatalog *journal.Catalog) *Server {
	if frontendFS == nil {
		frontendFS = os.DirFS(cfg.FrontendPath)
	}
	if bus == nil {
		bus = events.NewBus(logger)
	}
	if journalCatalog == nil {
		journalCatalog = journal.NewEmptyCatalog()
	}
	broadcaster := NewBroadcaster()
	broadcaster.RegisterSSESubscriber(bus)

	s := &Server{
		duck:             duck,
		sqlite:           sqlite,
		apex:             apexClient,
		cfg:              cfg,
		logger:           logger,
		broadcaster:      broadcaster,
		frontendFS:       frontendFS,
		catalog:          catalog,
		events:           bus,
		journalTemplates: journalCatalog,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Build middleware chain: RequestID → Logger → Recover → SecurityHeaders → CORS → Auth → handler
	var handler http.Handler = mux
	handler = Auth(sqlite)(handler)
	handler = CORS(handler)
	handler = SecurityHeaders(handler)
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
	// Health checks (unauthenticated).
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Probes.
	mux.HandleFunc("GET /api/probes", s.HandleProbeList)
	mux.HandleFunc("GET /api/probes/{name}/history", s.HandleProbeHistory)

	// Outlets.
	mux.HandleFunc("GET /api/outlets", s.HandleOutletList)
	mux.HandleFunc("GET /api/outlets/{id}/history", s.HandleOutletHistory)
	mux.HandleFunc("PUT /api/outlets/{id}", s.HandleOutletSet)

	// Programs (Apex output programs from /rest/config, stored in SQLite).
	mux.HandleFunc("GET /api/programs", s.HandleProgramList)
	mux.HandleFunc("POST /api/programs/sync", s.HandleProgramSync)

	// Feed mode.
	mux.HandleFunc("GET /api/feed", s.HandleFeedGet)
	mux.HandleFunc("PUT /api/feed", s.HandleFeedSet)

	// System.
	mux.HandleFunc("GET /api/system", s.HandleSystemStatus)

	// Devices.
	mux.HandleFunc("GET /api/devices", s.HandleDeviceList)
	mux.HandleFunc("GET /api/devices/suggestions", s.HandleDeviceSuggestions)
	mux.HandleFunc("POST /api/devices", s.HandleDeviceCreate)
	mux.HandleFunc("GET /api/devices/{id}", s.HandleDeviceGet)
	mux.HandleFunc("PUT /api/devices/{id}", s.HandleDeviceUpdate)
	mux.HandleFunc("DELETE /api/devices/{id}", s.HandleDeviceDelete)
	mux.HandleFunc("PUT /api/devices/{id}/probes", s.HandleDeviceSetProbes)
	mux.HandleFunc("PUT /api/devices/{id}/outlets", s.HandleDeviceSetOutlets)
	mux.HandleFunc("POST /api/devices/{id}/image", s.HandleDeviceImageUpload)
	mux.HandleFunc("DELETE /api/devices/{id}/image", s.HandleDeviceImageDelete)

	// Dashboard layout.
	mux.HandleFunc("GET /api/dashboard", s.HandleDashboardGet)
	mux.HandleFunc("PUT /api/dashboard", s.HandleDashboardReplace)
	mux.HandleFunc("POST /api/dashboard", s.HandleDashboardAddItem)
	mux.HandleFunc("DELETE /api/dashboard/{id}", s.HandleDashboardRemoveItem)

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
	mux.HandleFunc("GET /api/alerts/events", s.HandleAlertEvents)

	// Notifications.
	mux.HandleFunc("GET /api/notifications/targets", s.HandleNotificationTargetList)
	mux.HandleFunc("POST /api/notifications/targets", s.HandleNotificationTargetUpsert)
	mux.HandleFunc("DELETE /api/notifications/targets/{id}", s.HandleNotificationTargetDelete)
	mux.HandleFunc("POST /api/notifications/test", s.HandleNotificationTest)

	// System management.
	mux.HandleFunc("GET /api/system/log", s.HandleSystemLog)
	mux.HandleFunc("GET /api/system/backups", s.HandleBackupList)
	mux.HandleFunc("POST /api/system/backup", s.HandleBackupTrigger)
	mux.HandleFunc("POST /api/system/cleanup", s.HandleCleanup)

	// Export.
	mux.HandleFunc("GET /api/probes/{name}/export", s.HandleProbeExport)
	mux.HandleFunc("GET /api/export", s.HandleBulkExport)

	// Measurements.
	mux.HandleFunc("GET /api/measurements/parameters", s.HandleMeasurementParameterList)
	mux.HandleFunc("GET /api/measurements/kits", s.HandleKitList)
	mux.HandleFunc("GET /api/measurements", s.HandleMeasurementList)
	mux.HandleFunc("POST /api/measurements", s.HandleMeasurementCreate)
	mux.HandleFunc("PUT /api/measurements/{id}", s.HandleMeasurementUpdate)
	mux.HandleFunc("DELETE /api/measurements/{id}", s.HandleMeasurementDelete)

	// Livestock.
	mux.HandleFunc("GET /api/livestock", s.HandleLivestockList)
	mux.HandleFunc("GET /api/livestock/species", s.HandleLivestockSpecies) // must be before /{id}
	mux.HandleFunc("POST /api/livestock", s.HandleLivestockCreate)
	mux.HandleFunc("GET /api/livestock/{id}", s.HandleLivestockGet)
	mux.HandleFunc("PUT /api/livestock/{id}", s.HandleLivestockUpdate)
	mux.HandleFunc("DELETE /api/livestock/{id}", s.HandleLivestockDelete)
	mux.HandleFunc("POST /api/livestock/{id}/image", s.HandleLivestockImageUpload)
	mux.HandleFunc("POST /api/livestock/{id}/image/edit", s.HandleLivestockImageEdit)
	mux.HandleFunc("POST /api/livestock/{id}/image/reset", s.HandleLivestockImageReset)
	mux.HandleFunc("DELETE /api/livestock/{id}/image", s.HandleLivestockImageDelete)
	mux.HandleFunc("GET /api/livestock/{id}/observations", s.HandleLivestockObservationList)
	mux.HandleFunc("POST /api/livestock/{id}/observations", s.HandleLivestockObservationCreate)
	mux.HandleFunc("POST /api/livestock/{id}/observations/{obs_id}/image", s.HandleObservationImageUpload)
	mux.HandleFunc("DELETE /api/livestock/{id}/observations/{obs_id}/image", s.HandleObservationImageDelete)
	mux.HandleFunc("POST /api/livestock/{id}/observations/{obs_id}/image/edit", s.HandleObservationImageEdit)
	mux.HandleFunc("POST /api/livestock/{id}/observations/{obs_id}/image/reset", s.HandleObservationImageReset)

	// Tank profile.
	mux.HandleFunc("GET /api/tank/profile", s.HandleTankProfileGet)
	mux.HandleFunc("PUT /api/tank/profile/display", s.HandleTankProfileUpsert("display"))
	mux.HandleFunc("PUT /api/tank/profile/sump", s.HandleTankProfileUpsert("sump"))

	// Audit events.
	mux.HandleFunc("GET /api/events/stats", s.HandleEventBusStats) // must be before /api/events
	mux.HandleFunc("GET /api/events", s.HandleAuditEventList)

	// Daily prompt.
	mux.HandleFunc("GET /api/daily-prompt", s.HandleDailyPromptGet)
	mux.HandleFunc("POST /api/daily-prompt/respond", s.HandleDailyPromptRespond)

	// Journal.
	mux.HandleFunc("GET /api/journal/templates", s.HandleJournalTemplates) // must be before /{id}
	mux.HandleFunc("GET /api/journal", s.HandleJournalList)
	mux.HandleFunc("POST /api/journal", s.HandleJournalCreate)
	mux.HandleFunc("GET /api/journal/{id}", s.HandleJournalGet)
	mux.HandleFunc("PUT /api/journal/{id}", s.HandleJournalUpdate)
	mux.HandleFunc("DELETE /api/journal/{id}", s.HandleJournalDelete)

	// Dosing.
	mux.HandleFunc("GET /api/dosing/products", s.HandleDosingProductList)
	mux.HandleFunc("POST /api/dosing/products", s.HandleDosingProductCreate)
	mux.HandleFunc("PUT /api/dosing/products/{id}", s.HandleDosingProductUpdate)
	mux.HandleFunc("DELETE /api/dosing/products/{id}", s.HandleDosingProductDelete)
	mux.HandleFunc("GET /api/dosing/schedules", s.HandleDosingScheduleList)
	mux.HandleFunc("POST /api/dosing/schedules", s.HandleDosingScheduleCreate)
	mux.HandleFunc("PUT /api/dosing/schedules/{id}", s.HandleDosingScheduleUpdate)
	mux.HandleFunc("DELETE /api/dosing/schedules/{id}", s.HandleDosingScheduleDelete)
	mux.HandleFunc("POST /api/dosing/schedules/{id}/log", s.HandleDosingScheduleLog)
	mux.HandleFunc("GET /api/dosing/logs", s.HandleDosingLogList)

	// Maintenance.
	mux.HandleFunc("GET /api/maintenance/tasks", s.HandleMaintenanceTaskList)
	mux.HandleFunc("POST /api/maintenance/tasks", s.HandleMaintenanceTaskCreate)
	mux.HandleFunc("PUT /api/maintenance/tasks/{id}", s.HandleMaintenanceTaskUpdate)
	mux.HandleFunc("DELETE /api/maintenance/tasks/{id}", s.HandleMaintenanceTaskDelete)
	mux.HandleFunc("POST /api/maintenance/tasks/{id}/complete", s.HandleMaintenanceTaskComplete)
	mux.HandleFunc("GET /api/maintenance/tasks/{id}/logs", s.HandleMaintenanceLogList)

	// Due items (combined dosing + maintenance).
	mux.HandleFunc("GET /api/tasks/due", s.HandleDueItems)

	// Agent settings, context, and skills.
	mux.HandleFunc("GET /api/agent/settings", s.HandleAgentSettingsGet)
	mux.HandleFunc("PUT /api/agent/settings", s.HandleAgentSettingsPut)
	mux.HandleFunc("GET /api/agent/context", s.HandleAgentContext)
	mux.HandleFunc("GET /api/agent/skills", s.HandleAgentSkillList)
	mux.HandleFunc("GET /api/agent/skills/{name}/body", s.HandleAgentSkillBody)

	// HTTP MCP transport — requires SYMBIONT_TOKEN to create the loopback client.
	// Register each method explicitly; a bare "/api/mcp" pattern (no method prefix)
	// conflicts with "GET /" in Go 1.22's mux.
	if s.cfg.Token != "" {
		mcpHandler := newMCPHTTPHandler(s.cfg.APIPort, s.cfg.Token)
		mux.Handle("POST /api/mcp", mcpHandler)
		mux.Handle("GET /api/mcp", mcpHandler)
		mux.Handle("DELETE /api/mcp", mcpHandler)
	}

	// SSE stream.
	mux.HandleFunc("GET /api/stream", s.HandleStream)

	// Auth tokens.
	mux.HandleFunc("GET /api/tokens", s.HandleTokenList)
	mux.HandleFunc("POST /api/tokens", s.HandleTokenCreate)
	mux.HandleFunc("PATCH /api/tokens/{id}", s.HandleTokenUpdateScope)
	mux.HandleFunc("DELETE /api/tokens/{id}", s.HandleTokenDelete)

	// Image utilities.
	mux.HandleFunc("POST /api/images/reprocess", s.HandleImagesReprocess)

	// Device/livestock images — serve only the images subdirectory (not the whole data dir).
	// no-cache ensures the browser revalidates via If-Modified-Since after in-place overwrites.
	imagesDir := filepath.Join(filepath.Dir(s.sqlite.Path()), "images")
	imageFS := http.StripPrefix("/images/", http.FileServer(http.Dir(imagesDir)))
	mux.Handle("GET /images/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		imageFS.ServeHTTP(w, r)
	}))

	// Static frontend serving with SPA fallback.
	mux.Handle("GET /", spaHandler(s.frontendFS))
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	// Start background SSE poller.
	s.StartSSEPoller(ctx)

	// Synthetic request used to pass the server context into program sync helpers.
	syncReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

	// Seed programs on first run (no-op if DB already has records).
	s.SyncProgramsIfEmpty(syncReq)

	// Daily background sync to pick up any program changes.
	s.startDailyProgramSync(syncReq)

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

// Broadcaster returns the server's SSE broadcaster.
func (s *Server) Broadcaster() *Broadcaster {
	return s.broadcaster
}

// Addr returns the server's listener address. Only valid after Run has started.
func (s *Server) Addr() net.Addr {
	return nil // Will be useful for tests later if needed.
}

// spaHandler serves static files from fsys. If the requested file doesn't
// exist, it falls back to index.html to support client-side routing.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

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
