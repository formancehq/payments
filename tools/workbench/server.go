package workbench

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

//go:embed ui/dist/*
var uiFS embed.FS

// Server is the HTTP API server for the workbench.
type Server struct {
	workbench *Workbench
	addr      string
	enableUI  bool
	server    *http.Server
}

// NewServer creates a new workbench HTTP server.
func NewServer(wb *Workbench, addr string, enableUI bool) *Server {
	return &Server{
		workbench: wb,
		addr:      addr,
		enableUI:  enableUI,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", s.handleStatus)
		r.Post("/install", s.handleInstall)
		r.Post("/uninstall", s.handleUninstall)
		r.Post("/reset", s.handleReset)

		// Fetch operations
		r.Post("/fetch/accounts", s.handleFetchAccounts)
		r.Post("/fetch/payments", s.handleFetchPayments)
		r.Post("/fetch/balances", s.handleFetchBalances)
		r.Post("/fetch/all", s.handleFetchAll)

		// Write operations
		r.Post("/transfer", s.handleCreateTransfer)
		r.Post("/payout", s.handleCreatePayout)

		// Data endpoints
		r.Get("/data/accounts", s.handleGetAccounts)
		r.Get("/data/payments", s.handleGetPayments)
		r.Get("/data/balances", s.handleGetBalances)
		r.Get("/data/external-accounts", s.handleGetExternalAccounts)
		r.Get("/data/others", s.handleGetOthers)
		r.Get("/data/states", s.handleGetStates)
		r.Get("/data/tasks-tree", s.handleGetTasksTree)
		r.Get("/data/export", s.handleExport)
		r.Post("/data/import", s.handleImport)

		// Debug endpoints
		r.Get("/debug/logs", s.handleDebugLogs)
		r.Get("/debug/requests", s.handleDebugRequests)
		r.Get("/debug/plugin-calls", s.handleDebugPluginCalls)
		r.Get("/debug/stats", s.handleDebugStats)
		r.Delete("/debug/clear", s.handleDebugClear)

		// HTTP capture control
		r.Get("/debug/http-capture", s.handleHTTPCaptureStatus)
		r.Post("/debug/http-capture/enable", s.handleHTTPCaptureEnable)
		r.Post("/debug/http-capture/disable", s.handleHTTPCaptureDisable)

		// Config endpoints
		r.Get("/config", s.handleGetConfig)
		r.Put("/config/page-size", s.handleSetPageSize)

		// Introspection endpoints
		r.Get("/introspect/info", s.handleIntrospectInfo)
		r.Get("/introspect/files", s.handleIntrospectFiles)
		r.Get("/introspect/file", s.handleIntrospectFile)
		r.Get("/introspect/search", s.handleIntrospectSearch)

		// Task tracking endpoints
		r.Get("/tasks", s.handleGetTasks)
		r.Get("/tasks/executions", s.handleGetTaskExecutions)
		r.Post("/tasks/step", s.handleTaskStep)
		r.Put("/tasks/step-mode", s.handleSetStepMode)
		r.Post("/tasks/reset", s.handleResetTasks)

		// Replay endpoints
		r.Get("/replay/history", s.handleReplayHistory)
		r.Get("/replay/{id}", s.handleGetReplay)
		r.Post("/replay", s.handleReplay)
		r.Post("/replay/from-capture/{id}", s.handleReplayFromCapture)
		r.Get("/replay/compare/{id}", s.handleReplayCompare)
		r.Post("/replay/dry-run", s.handleReplayDryRun)
		r.Get("/replay/curl/{id}", s.handleReplayCurl)
		r.Delete("/replay/history", s.handleClearReplayHistory)

		// Snapshot endpoints
		r.Get("/snapshots", s.handleListSnapshots)
		r.Get("/snapshots/stats", s.handleSnapshotStats)
		r.Get("/snapshots/{id}", s.handleGetSnapshot)
		r.Post("/snapshots", s.handleCreateSnapshot)
		r.Post("/snapshots/from-capture/{id}", s.handleCreateSnapshotFromCapture)
		r.Delete("/snapshots/{id}", s.handleDeleteSnapshot)
		r.Delete("/snapshots", s.handleClearSnapshots)
		r.Post("/snapshots/export", s.handleExportSnapshots)
		r.Post("/snapshots/import", s.handleImportSnapshots)

		// Test generation endpoints
		r.Get("/tests/preview", s.handlePreviewTests)
		r.Post("/tests/generate", s.handleGenerateTests)

		// Schema inference endpoints
		r.Get("/schemas", s.handleListSchemas)
		r.Get("/schemas/stats", s.handleSchemaStats)
		r.Get("/schemas/{operation}", s.handleGetSchema)
		r.Post("/schemas/infer", s.handleInferSchema)
		r.Post("/schemas/baselines", s.handleSaveSchemaBaseline)
		r.Post("/schemas/baselines/all", s.handleSaveAllSchemaBaselines)
		r.Get("/schemas/baselines", s.handleListSchemaBaselines)
		r.Get("/schemas/compare/{operation}", s.handleCompareSchema)
		r.Delete("/schemas", s.handleClearSchemas)

		// Data baseline endpoints
		r.Get("/baselines", s.handleListBaselines)
		r.Get("/baselines/{id}", s.handleGetBaseline)
		r.Post("/baselines", s.handleSaveBaseline)
		r.Delete("/baselines/{id}", s.handleDeleteBaseline)
		r.Get("/baselines/{id}/compare", s.handleCompareBaseline)
		r.Get("/baselines/{id}/export", s.handleExportBaseline)
		r.Post("/baselines/import", s.handleImportBaseline)
	})

	// Serve embedded UI
	if s.enableUI {
		// Try to serve from embedded filesystem
		uiContent, err := fs.Sub(uiFS, "ui/dist")
		if err == nil {
			// Serve static assets
			fileServer := http.FileServer(http.FS(uiContent))
			
			// Handle /ui/assets/* for static files
			r.Handle("/ui/assets/*", http.StripPrefix("/ui", fileServer))
			
			// Handle /ui and /ui/ to serve index.html
			r.Get("/ui", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
			})
			r.Get("/ui/", func(w http.ResponseWriter, req *http.Request) {
				indexHTML, err := fs.ReadFile(uiContent, "index.html")
				if err != nil {
					http.Error(w, "UI not available", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Write(indexHTML)
			})
		}

		// Root redirects to /ui/ if UI is enabled, otherwise fallback
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			if uiContent, err := fs.Sub(uiFS, "ui/dist"); err == nil {
				if _, err := fs.ReadFile(uiContent, "index.html"); err == nil {
					http.Redirect(w, req, "/ui/", http.StatusTemporaryRedirect)
					return
				}
			}
			s.handleUIFallback(w, req)
		})
	} else {
		r.Get("/", s.handleUIFallback)
	}

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: r,
	}

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// === Helper functions ===

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// === Status ===

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	storage := s.workbench.Storage()
	debug := s.workbench.Debug()
	engine := s.workbench.Engine()

	stats := storage.GetStats()
	debugStats := debug.Stats()
	fetchStatus := engine.GetFetchStatus()

	connectorID := s.workbench.ConnectorID()
	status := map[string]interface{}{
		"connector_id": connectorID.String(),
		"provider":     s.workbench.Config().Provider,
		"storage": map[string]interface{}{
			"accounts_count":          stats.AccountsCount,
			"payments_count":          stats.PaymentsCount,
			"balances_count":          stats.BalancesCount,
			"external_accounts_count": stats.ExternalAccountsCount,
			"last_updated":            stats.LastUpdated,
		},
		"debug": debugStats,
		"fetch_status": fetchStatus,
	}

	s.jsonResponse(w, http.StatusOK, status)
}

// === Install/Uninstall ===

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	if err := s.workbench.Engine().Install(r.Context()); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "installed"})
}

func (s *Server) handleUninstall(w http.ResponseWriter, r *http.Request) {
	if err := s.workbench.Engine().Uninstall(r.Context()); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "uninstalled"})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	s.workbench.Engine().ResetFetchState()
	s.workbench.Storage().Clear()
	s.workbench.Debug().Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "reset"})
}

// === Fetch Operations ===

type fetchRequest struct {
	FromPayload json.RawMessage `json:"from_payload,omitempty"`
}

func (s *Server) handleFetchAccounts(w http.ResponseWriter, r *http.Request) {
	var req fetchRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	resp, err := s.workbench.Engine().FetchAccountsOnePage(r.Context(), req.FromPayload)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"accounts": resp.Accounts,
		"has_more": resp.HasMore,
		"count":    len(resp.Accounts),
	})
}

func (s *Server) handleFetchPayments(w http.ResponseWriter, r *http.Request) {
	var req fetchRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	resp, err := s.workbench.Engine().FetchPaymentsOnePage(r.Context(), req.FromPayload)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"payments": resp.Payments,
		"has_more": resp.HasMore,
		"count":    len(resp.Payments),
	})
}

func (s *Server) handleFetchBalances(w http.ResponseWriter, r *http.Request) {
	var req fetchRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	resp, err := s.workbench.Engine().FetchBalancesOnePage(r.Context(), req.FromPayload)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"balances": resp.Balances,
		"has_more": resp.HasMore,
		"count":    len(resp.Balances),
	})
}

func (s *Server) handleFetchAll(w http.ResponseWriter, r *http.Request) {
	if err := s.workbench.Engine().RunOneCycle(r.Context()); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats := s.workbench.Storage().GetStats()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "complete",
		"accounts": stats.AccountsCount,
		"payments": stats.PaymentsCount,
		"balances": stats.BalancesCount,
	})
}

// === Write Operations ===

func (s *Server) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse transfer request from body
	s.errorResponse(w, http.StatusNotImplemented, "transfer creation not yet implemented in workbench")
}

func (s *Server) handleCreatePayout(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse payout request from body
	s.errorResponse(w, http.StatusNotImplemented, "payout creation not yet implemented in workbench")
}

// === Data Endpoints ===

func (s *Server) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := s.workbench.Storage().GetAccounts()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

func (s *Server) handleGetPayments(w http.ResponseWriter, r *http.Request) {
	payments := s.workbench.Storage().GetPayments()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"payments": payments,
		"count":    len(payments),
	})
}

func (s *Server) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	accountRef := r.URL.Query().Get("account")
	var balances interface{}

	if accountRef != "" {
		balances = s.workbench.Storage().GetBalancesForAccount(accountRef)
	} else {
		balances = s.workbench.Storage().GetBalances()
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"balances": balances,
	})
}

func (s *Server) handleGetExternalAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := s.workbench.Storage().GetExternalAccounts()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"external_accounts": accounts,
		"count":             len(accounts),
	})
}

func (s *Server) handleGetOthers(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	var result interface{}

	if name != "" {
		result = s.workbench.Storage().GetOthers(name)
	} else {
		result = s.workbench.Storage().GetAllOthers()
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"others": result,
	})
}

func (s *Server) handleGetStates(w http.ResponseWriter, r *http.Request) {
	states := s.workbench.Storage().GetAllStates()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"states": states,
	})
}

func (s *Server) handleGetTasksTree(w http.ResponseWriter, r *http.Request) {
	tree := s.workbench.Storage().GetTasksTree()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"tasks_tree": tree,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	snapshot := s.workbench.Storage().Export()
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=workbench-export-%s.json", time.Now().Format("20060102-150405")))
	json.NewEncoder(w).Encode(snapshot)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var snapshot StorageSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid snapshot format")
		return
	}

	s.workbench.Storage().Import(snapshot)
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "imported"})
}

// === Debug Endpoints ===

func (s *Server) handleDebugLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	typeStr := r.URL.Query().Get("type")
	var entryType *DebugEntryType
	if typeStr != "" {
		t := DebugEntryType(typeStr)
		entryType = &t
	}

	entries := s.workbench.Debug().GetEntries(entryType, limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

func (s *Server) handleDebugRequests(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	requests := s.workbench.Debug().GetHTTPRequests(limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	})
}

func (s *Server) handleDebugPluginCalls(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	calls := s.workbench.Debug().GetPluginCalls(limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"plugin_calls": calls,
		"count":        len(calls),
	})
}

func (s *Server) handleDebugStats(w http.ResponseWriter, r *http.Request) {
	stats := s.workbench.Debug().Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleDebugClear(w http.ResponseWriter, r *http.Request) {
	s.workbench.Debug().Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleHTTPCaptureStatus(w http.ResponseWriter, r *http.Request) {
	transport := s.workbench.Transport()
	enabled := false
	if transport != nil {
		enabled = transport.IsEnabled()
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"enabled":       enabled,
		"max_body_size": transport.MaxBodySize,
	})
}

func (s *Server) handleHTTPCaptureEnable(w http.ResponseWriter, r *http.Request) {
	transport := s.workbench.Transport()
	if transport != nil {
		transport.Enable()
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (s *Server) handleHTTPCaptureDisable(w http.ResponseWriter, r *http.Request) {
	transport := s.workbench.Transport()
	if transport != nil {
		transport.Disable()
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// === Config Endpoints ===

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	connectorID := s.workbench.ConnectorID()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"provider":      s.workbench.Config().Provider,
		"connector_id":  connectorID.String(),
		"listen_addr":   s.workbench.Config().ListenAddr,
		"auto_poll":     s.workbench.Config().AutoPoll,
		"poll_interval": s.workbench.Config().PollInterval.String(),
	})
}

func (s *Server) handleSetPageSize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PageSize int `json:"page_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.PageSize < 1 || req.PageSize > 1000 {
		s.errorResponse(w, http.StatusBadRequest, "page_size must be between 1 and 1000")
		return
	}

	s.workbench.Engine().SetPageSize(req.PageSize)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"page_size": req.PageSize})
}

// === Introspection Endpoints ===

func (s *Server) handleIntrospectInfo(w http.ResponseWriter, r *http.Request) {
	intro := s.workbench.Introspector()
	if intro == nil {
		s.errorResponse(w, http.StatusInternalServerError, "introspector not available")
		return
	}

	info, err := intro.GetInfo()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, info)
}

func (s *Server) handleIntrospectFiles(w http.ResponseWriter, r *http.Request) {
	intro := s.workbench.Introspector()
	if intro == nil {
		s.errorResponse(w, http.StatusInternalServerError, "introspector not available")
		return
	}

	files, err := intro.GetFileTree()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"files": files})
}

func (s *Server) handleIntrospectFile(w http.ResponseWriter, r *http.Request) {
	intro := s.workbench.Introspector()
	if intro == nil {
		s.errorResponse(w, http.StatusInternalServerError, "introspector not available")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		s.errorResponse(w, http.StatusBadRequest, "path parameter required")
		return
	}

	file, err := intro.GetFile(path)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, file)
}

func (s *Server) handleIntrospectSearch(w http.ResponseWriter, r *http.Request) {
	intro := s.workbench.Introspector()
	if intro == nil {
		s.errorResponse(w, http.StatusInternalServerError, "introspector not available")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.errorResponse(w, http.StatusBadRequest, "q parameter required")
		return
	}

	results, err := intro.SearchCode(query)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// === Task Tracking Endpoints ===

func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.workbench.Tasks()
	if tasks == nil {
		s.errorResponse(w, http.StatusInternalServerError, "task tracker not available")
		return
	}

	summary := tasks.GetSummary()
	summary.IsRunning = s.workbench.Engine().IsRunning()
	s.jsonResponse(w, http.StatusOK, summary)
}

func (s *Server) handleGetTaskExecutions(w http.ResponseWriter, r *http.Request) {
	tasks := s.workbench.Tasks()
	if tasks == nil {
		s.errorResponse(w, http.StatusInternalServerError, "task tracker not available")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	executions := tasks.GetExecutions(limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}

func (s *Server) handleTaskStep(w http.ResponseWriter, r *http.Request) {
	tasks := s.workbench.Tasks()
	if tasks == nil {
		s.errorResponse(w, http.StatusInternalServerError, "task tracker not available")
		return
	}

	tasks.Step()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "stepped"})
}

func (s *Server) handleSetStepMode(w http.ResponseWriter, r *http.Request) {
	tasks := s.workbench.Tasks()
	if tasks == nil {
		s.errorResponse(w, http.StatusInternalServerError, "task tracker not available")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request")
		return
	}

	tasks.SetStepMode(req.Enabled)
	s.jsonResponse(w, http.StatusOK, map[string]bool{"step_mode": req.Enabled})
}

func (s *Server) handleResetTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.workbench.Tasks()
	if tasks == nil {
		s.errorResponse(w, http.StatusInternalServerError, "task tracker not available")
		return
	}

	tasks.Reset()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "reset"})
}

// === Replay Endpoints ===

func (s *Server) handleReplayHistory(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	history := replayer.GetHistory(limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

func (s *Server) handleGetReplay(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	id := chi.URLParam(r, "id")
	replay := replayer.GetReplayByID(id)
	if replay == nil {
		s.errorResponse(w, http.StatusNotFound, "replay not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, replay)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	var req ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	resp, err := replayer.Replay(r.Context(), req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleReplayFromCapture(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	captureID := chi.URLParam(r, "id")

	// Parse optional modifications
	var modifications *ReplayRequest
	if r.ContentLength > 0 {
		modifications = &ReplayRequest{}
		if err := json.NewDecoder(r.Body).Decode(modifications); err != nil {
			s.errorResponse(w, http.StatusBadRequest, "invalid modifications: "+err.Error())
			return
		}
	}

	resp, err := replayer.ReplayFromCapture(r.Context(), captureID, modifications)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleReplayCompare(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	replayID := chi.URLParam(r, "id")
	comparison, err := replayer.Compare(replayID)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, comparison)
}

func (s *Server) handleReplayDryRun(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	var req ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	result, err := replayer.DryRun(r.Context(), req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleReplayCurl(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	debug := s.workbench.Debug()
	if replayer == nil || debug == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	id := chi.URLParam(r, "id")
	
	// Try to find in captured requests
	captured := debug.GetHTTPRequestByID(id)
	if captured != nil {
		req := ReplayRequest{
			Method:  captured.Method,
			URL:     captured.URL,
			Headers: captured.RequestHeaders,
			Body:    captured.RequestBody,
		}
		curl := replayer.CreateCurlCommand(req)
		s.jsonResponse(w, http.StatusOK, map[string]string{"curl": curl})
		return
	}

	// Try to find in replay history
	replay := replayer.GetReplayByID(id)
	if replay != nil {
		curl := replayer.CreateCurlCommand(replay.Request)
		s.jsonResponse(w, http.StatusOK, map[string]string{"curl": curl})
		return
	}

	s.errorResponse(w, http.StatusNotFound, "request not found")
}

func (s *Server) handleClearReplayHistory(w http.ResponseWriter, r *http.Request) {
	replayer := s.workbench.Replayer()
	if replayer == nil {
		s.errorResponse(w, http.StatusInternalServerError, "replayer not available")
		return
	}

	replayer.ClearHistory()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// === Snapshot Endpoints ===

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	operation := r.URL.Query().Get("operation")
	tagsParam := r.URL.Query().Get("tags")
	var tags []string
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	list := snapshots.List(operation, tags)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"snapshots": list,
		"count":     len(list),
	})
}

func (s *Server) handleSnapshotStats(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	stats := snapshots.Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	id := chi.URLParam(r, "id")
	snapshot := snapshots.Get(id)
	if snapshot == nil {
		s.errorResponse(w, http.StatusNotFound, "snapshot not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, snapshot)
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	var snapshot Snapshot
	if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := snapshots.Save(&snapshot); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, snapshot)
}

func (s *Server) handleCreateSnapshotFromCapture(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	captureID := chi.URLParam(r, "id")

	var req struct {
		Name        string   `json:"name"`
		Operation   string   `json:"operation"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Operation == "" {
		s.errorResponse(w, http.StatusBadRequest, "operation is required")
		return
	}

	snapshot, err := snapshots.SaveFromCapture(captureID, req.Name, req.Operation, req.Description, req.Tags)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, snapshot)
}

func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	id := chi.URLParam(r, "id")
	if err := snapshots.Delete(id); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleClearSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	snapshots.Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleExportSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	var req struct {
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Directory == "" {
		s.errorResponse(w, http.StatusBadRequest, "directory is required")
		return
	}

	if err := snapshots.ExportToDir(req.Directory); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "exported",
		"directory": req.Directory,
	})
}

func (s *Server) handleImportSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots := s.workbench.Snapshots()
	if snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "snapshots not available")
		return
	}

	var req struct {
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Directory == "" {
		s.errorResponse(w, http.StatusBadRequest, "directory is required")
		return
	}

	count, err := snapshots.ImportFromDir(req.Directory)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "imported",
		"imported": count,
	})
}

// === Test Generation Endpoints ===

func (s *Server) handlePreviewTests(w http.ResponseWriter, r *http.Request) {
	testGen := s.workbench.TestGenerator()
	if testGen == nil {
		s.errorResponse(w, http.StatusInternalServerError, "test generator not available")
		return
	}

	result, err := testGen.PreviewTest()
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleGenerateTests(w http.ResponseWriter, r *http.Request) {
	testGen := s.workbench.TestGenerator()
	snapshots := s.workbench.Snapshots()
	if testGen == nil || snapshots == nil {
		s.errorResponse(w, http.StatusInternalServerError, "test generator not available")
		return
	}

	var req struct {
		OutputDir string `json:"output_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	result, err := testGen.Generate()
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// If output directory specified, write files
	if req.OutputDir != "" {
		if err := writeGeneratedFiles(req.OutputDir, result); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		result.Instructions = fmt.Sprintf("Files written to %s\n\n%s", req.OutputDir, result.Instructions)
	}

	s.jsonResponse(w, http.StatusOK, result)
}

// === Fallback UI ===

func (s *Server) handleUIFallback(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Connector Workbench</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        h1 { color: #58a6ff; margin-bottom: 20px; }
        h2 { color: #8b949e; font-size: 14px; text-transform: uppercase; margin: 20px 0 10px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: #161b22; border: 1px solid #30363d; border-radius: 6px; padding: 16px; }
        .card-title { color: #58a6ff; font-size: 14px; font-weight: 600; margin-bottom: 12px; }
        .stat { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #21262d; }
        .stat:last-child { border-bottom: none; }
        .stat-label { color: #8b949e; }
        .stat-value { color: #c9d1d9; font-weight: 500; }
        .btn { background: #238636; border: none; color: white; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 14px; margin: 4px; }
        .btn:hover { background: #2ea043; }
        .btn-secondary { background: #21262d; border: 1px solid #30363d; }
        .btn-secondary:hover { background: #30363d; }
        .btn-danger { background: #da3633; }
        .btn-danger:hover { background: #f85149; }
        .actions { margin: 20px 0; }
        pre { background: #0d1117; border: 1px solid #30363d; border-radius: 6px; padding: 12px; overflow-x: auto; font-size: 12px; max-height: 400px; overflow-y: auto; }
        .logs { font-family: monospace; font-size: 11px; }
        .log-entry { padding: 4px 0; border-bottom: 1px solid #21262d; }
        .log-time { color: #8b949e; }
        .log-type { padding: 2px 6px; border-radius: 3px; font-size: 10px; margin: 0 8px; }
        .log-type-log { background: #238636; }
        .log-type-error { background: #da3633; }
        .log-type-plugin_call { background: #1f6feb; }
        .log-type-state_change { background: #a371f7; }
        #status { margin-top: 10px; padding: 10px; background: #21262d; border-radius: 6px; }
        .refresh-hint { color: #8b949e; font-size: 12px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 Connector Workbench</h1>
        
        <div class="actions">
            <button class="btn" onclick="fetchAll()">▶️ Run Full Cycle</button>
            <button class="btn btn-secondary" onclick="fetchAccounts()">Fetch Accounts</button>
            <button class="btn btn-secondary" onclick="fetchPayments()">Fetch Payments</button>
            <button class="btn btn-secondary" onclick="fetchBalances()">Fetch Balances</button>
            <button class="btn btn-danger" onclick="reset()">🔄 Reset</button>
        </div>
        
        <div id="status"></div>

        <h2>Data</h2>
        <div class="grid">
            <div class="card">
                <div class="card-title">Accounts</div>
                <div id="accounts-data">Loading...</div>
            </div>
            <div class="card">
                <div class="card-title">Payments</div>
                <div id="payments-data">Loading...</div>
            </div>
            <div class="card">
                <div class="card-title">Balances</div>
                <div id="balances-data">Loading...</div>
            </div>
        </div>

        <h2>Debug Logs</h2>
        <div class="card">
            <div class="card-title">Recent Activity</div>
            <div id="logs" class="logs">Loading...</div>
        </div>

        <h2>Plugin Calls</h2>
        <div class="card">
            <pre id="plugin-calls">Loading...</pre>
        </div>

        <p class="refresh-hint">Data refreshes every 2 seconds</p>
    </div>

    <script>
        const API = '/api';

        async function fetchJSON(url, opts = {}) {
            const res = await fetch(API + url, opts);
            return res.json();
        }

        async function fetchAll() {
            updateStatus('Running full fetch cycle...');
            const res = await fetchJSON('/fetch/all', { method: 'POST' });
            updateStatus('Cycle complete: ' + JSON.stringify(res));
            refresh();
        }

        async function fetchAccounts() {
            updateStatus('Fetching accounts...');
            const res = await fetchJSON('/fetch/accounts', { method: 'POST' });
            updateStatus('Fetched ' + res.count + ' accounts (has_more: ' + res.has_more + ')');
            refresh();
        }

        async function fetchPayments() {
            updateStatus('Fetching payments...');
            const res = await fetchJSON('/fetch/payments', { method: 'POST' });
            updateStatus('Fetched ' + res.count + ' payments (has_more: ' + res.has_more + ')');
            refresh();
        }

        async function fetchBalances() {
            updateStatus('Fetching balances...');
            const res = await fetchJSON('/fetch/balances', { method: 'POST' });
            updateStatus('Fetched ' + res.count + ' balances (has_more: ' + res.has_more + ')');
            refresh();
        }

        async function reset() {
            if (confirm('Reset all data and state?')) {
                await fetchJSON('/reset', { method: 'POST' });
                updateStatus('Reset complete');
                refresh();
            }
        }

        function updateStatus(msg) {
            document.getElementById('status').textContent = msg;
        }

        async function refresh() {
            // Accounts
            const accounts = await fetchJSON('/data/accounts');
            document.getElementById('accounts-data').innerHTML = 
                '<div class="stat"><span class="stat-label">Count</span><span class="stat-value">' + accounts.count + '</span></div>' +
                (accounts.accounts.slice(0, 5).map(a => 
                    '<div class="stat"><span class="stat-label">' + a.reference + '</span><span class="stat-value">' + (a.name || '-') + '</span></div>'
                ).join(''));

            // Payments
            const payments = await fetchJSON('/data/payments');
            document.getElementById('payments-data').innerHTML = 
                '<div class="stat"><span class="stat-label">Count</span><span class="stat-value">' + payments.count + '</span></div>' +
                (payments.payments.slice(0, 5).map(p => 
                    '<div class="stat"><span class="stat-label">' + p.reference + '</span><span class="stat-value">' + p.amount + ' ' + p.asset + '</span></div>'
                ).join(''));

            // Balances
            const balances = await fetchJSON('/data/balances');
            const balancesList = balances.balances || [];
            document.getElementById('balances-data').innerHTML = 
                '<div class="stat"><span class="stat-label">Count</span><span class="stat-value">' + balancesList.length + '</span></div>' +
                (balancesList.slice(0, 5).map(b => 
                    '<div class="stat"><span class="stat-label">' + b.account_reference + '</span><span class="stat-value">' + b.amount + ' ' + b.asset + '</span></div>'
                ).join(''));

            // Logs
            const logs = await fetchJSON('/debug/logs?limit=20');
            document.getElementById('logs').innerHTML = logs.entries.map(e => 
                '<div class="log-entry"><span class="log-time">' + new Date(e.timestamp).toLocaleTimeString() + '</span>' +
                '<span class="log-type log-type-' + e.type + '">' + e.type + '</span>' +
                '<span>' + (e.operation || '') + ' ' + (e.message || '') + (e.error ? ' ERROR: ' + e.error : '') + '</span></div>'
            ).join('');

            // Plugin calls
            const calls = await fetchJSON('/debug/plugin-calls?limit=10');
            document.getElementById('plugin-calls').textContent = JSON.stringify(calls.plugin_calls, null, 2);
        }

        // Initial load
        refresh();
        // Auto-refresh
        setInterval(refresh, 2000);
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// === Schema Inference Endpoints ===

func (s *Server) handleListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	list := schemas.ListSchemas()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"schemas": list,
		"count":   len(list),
	})
}

func (s *Server) handleSchemaStats(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	stats := schemas.Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	operation := chi.URLParam(r, "operation")
	schema := schemas.GetSchema(operation)
	if schema == nil {
		s.errorResponse(w, http.StatusNotFound, "schema not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, schema)
}

func (s *Server) handleInferSchema(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	var req struct {
		Operation string          `json:"operation"`
		Endpoint  string          `json:"endpoint"`
		Method    string          `json:"method"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	schema, err := schemas.InferFromJSON(req.Operation, req.Endpoint, req.Method, req.Data)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, schema)
}

func (s *Server) handleSaveSchemaBaseline(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	var req struct {
		Operation string `json:"operation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := schemas.SaveBaseline(req.Operation); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handleSaveAllSchemaBaselines(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	count := schemas.SaveAllBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "saved",
		"count":  count,
	})
}

func (s *Server) handleListSchemaBaselines(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	list := schemas.ListBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"baselines": list,
		"count":     len(list),
	})
}

func (s *Server) handleCompareSchema(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	operation := chi.URLParam(r, "operation")
	diff, err := schemas.CompareWithBaseline(operation)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, diff)
}

func (s *Server) handleClearSchemas(w http.ResponseWriter, r *http.Request) {
	schemas := s.workbench.Schemas()
	if schemas == nil {
		s.errorResponse(w, http.StatusInternalServerError, "schema manager not available")
		return
	}

	schemas.Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// === Data Baseline Endpoints ===

func (s *Server) handleListBaselines(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	list := baselines.ListBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"baselines": list,
		"count":     len(list),
	})
}

func (s *Server) handleGetBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	baseline := baselines.GetBaseline(id)
	if baseline == nil {
		s.errorResponse(w, http.StatusNotFound, "baseline not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, baseline)
}

func (s *Server) handleSaveBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("baseline-%s", time.Now().Format("2006-01-02-15-04"))
	}

	baseline, err := baselines.SaveBaseline(req.Name)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, baseline)
}

func (s *Server) handleDeleteBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	if err := baselines.DeleteBaseline(id); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleCompareBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	diff, err := baselines.CompareWithCurrent(id)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, diff)
}

func (s *Server) handleExportBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	data, err := baselines.Export(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", id))
	w.Write(data)
}

func (s *Server) handleImportBaseline(w http.ResponseWriter, r *http.Request) {
	baselines := s.workbench.Baselines()
	if baselines == nil {
		s.errorResponse(w, http.StatusInternalServerError, "baseline manager not available")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	baseline, err := baselines.Import(data)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, baseline)
}

// writeGeneratedFiles writes generated test files to the specified directory.
func writeGeneratedFiles(outputDir string, result *GenerateResult) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create testdata subdirectory
	testdataDir := filepath.Join(outputDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		return fmt.Errorf("failed to create testdata directory: %w", err)
	}

	// Write test file
	testPath := filepath.Join(outputDir, result.TestFile.Filename)
	if err := os.WriteFile(testPath, []byte(result.TestFile.Content), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	// Write fixture files
	for _, fixture := range result.Fixtures {
		fixturePath := filepath.Join(testdataDir, fixture.Filename)
		if err := os.WriteFile(fixturePath, []byte(fixture.Content), 0644); err != nil {
			return fmt.Errorf("failed to write fixture %s: %w", fixture.Filename, err)
		}
	}

	return nil
}
