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

	"github.com/formancehq/payments/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
		// Global status
		r.Get("/status", s.handleGlobalStatus)

		// Connector management
		r.Get("/connectors/available", s.handleAvailableConnectors)
		r.Get("/connectors", s.handleListConnectors)
		r.Post("/connectors", s.handleCreateConnector)

		// Connector-specific routes
		r.Route("/connectors/{connectorID}", func(r chi.Router) {
			r.Use(s.connectorCtx) // Middleware to load connector instance

			r.Get("/", s.handleConnectorStatus)
			r.Delete("/", s.handleDeleteConnector)
			r.Post("/install", s.handleInstall)
			r.Post("/uninstall", s.handleUninstall)
			r.Post("/reset", s.handleReset)

			// Open banking operations
			r.Post("/ob/create-user", s.handleOBCreateUser)
			r.Post("/ob/create-link", s.handleOBCreateUserLink)
			r.Post("/ob/complete-link", s.handleOBCompleteUserLink)
			r.Get("/ob/connections", s.handleOBListConnections)
			r.Delete("/ob/connections/{connectionID}", s.handleOBDeleteConnection)

			// Fetch operations
			r.Post("/fetch/accounts", s.handleFetchAccounts)
			r.Post("/fetch/payments", s.handleFetchPayments)
			r.Post("/fetch/balances", s.handleFetchBalances)
			r.Post("/fetch/external-accounts", s.handleFetchExternalAccounts)
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

		// Global debug endpoints (shared across all connectors)
		r.Get("/debug/logs", s.handleDebugLogs)
		r.Get("/debug/requests", s.handleDebugRequests)
		r.Get("/debug/plugin-calls", s.handleDebugPluginCalls)
		r.Get("/debug/stats", s.handleDebugStats)
		r.Delete("/debug/clear", s.handleDebugClear)

		// HTTP capture control
		r.Get("/debug/http-capture", s.handleHTTPCaptureStatus)
		r.Post("/debug/http-capture/enable", s.handleHTTPCaptureEnable)
		r.Post("/debug/http-capture/disable", s.handleHTTPCaptureDisable)

		// Replay endpoints (global)
		r.Get("/replay/history", s.handleReplayHistory)
		r.Get("/replay/{id}", s.handleGetReplay)
		r.Post("/replay", s.handleReplay)
		r.Post("/replay/from-capture/{id}", s.handleReplayFromCapture)
		r.Get("/replay/compare/{id}", s.handleReplayCompare)
		r.Post("/replay/dry-run", s.handleReplayDryRun)
		r.Get("/replay/curl/{id}", s.handleReplayCurl)
		r.Delete("/replay/history", s.handleClearReplayHistory)

		// Generic server configuration
		r.Get("/generic-server/status", s.handleGenericServerStatus)
		r.Post("/generic-server/connector", s.handleSetGenericConnector)
	})

	// Generic connector server (for remote integration testing)
	// Exposes the standard generic connector API so staging services can connect
	r.Route("/generic", func(r chi.Router) {
		r.Use(s.genericServerMiddleware)
		r.Get("/accounts", s.handleGenericAccounts)
		r.Get("/accounts/{accountId}/balances", s.handleGenericBalances)
		r.Get("/beneficiaries", s.handleGenericBeneficiaries)
		r.Get("/transactions", s.handleGenericTransactions)
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
				_, _ = w.Write(indexHTML)
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

// === Context Key for Connector Instance ===

type ctxKey string

const connectorCtxKey ctxKey = "connector"

// connectorCtx middleware loads the connector instance from URL param
func (s *Server) connectorCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectorID := chi.URLParam(r, "connectorID")
		if connectorID == "" {
			s.errorResponse(w, http.StatusBadRequest, "connector ID is required")
			return
		}

		conn := s.workbench.GetConnector(connectorID)
		if conn == nil {
			s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("connector %s not found", connectorID))
			return
		}

		ctx := context.WithValue(r.Context(), connectorCtxKey, conn)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getConnector retrieves the connector from request context
func (s *Server) getConnector(r *http.Request) *ConnectorInstance {
	conn, _ := r.Context().Value(connectorCtxKey).(*ConnectorInstance)
	return conn
}

// === Helper functions ===

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// === Global Status ===

func (s *Server) handleGlobalStatus(w http.ResponseWriter, r *http.Request) {
	connectors := s.workbench.ListConnectors()
	debugStats := s.workbench.Debug().Stats()

	connectorSummaries := make([]map[string]interface{}, 0, len(connectors))
	for _, conn := range connectors {
		summary := map[string]interface{}{
			"id":           conn.ID,
			"provider":     conn.Provider,
			"name":         conn.Name,
			"connector_id": conn.ConnectorID.String(),
			"installed":    conn.Installed,
			"created_at":   conn.CreatedAt,
		}
		if conn.storage != nil {
			stats := conn.storage.GetStats()
			summary["accounts_count"] = stats.AccountsCount
			summary["payments_count"] = stats.PaymentsCount
			summary["balances_count"] = stats.BalancesCount
		}
		connectorSummaries = append(connectorSummaries, summary)
	}

	status := map[string]interface{}{
		"connectors_count": len(connectors),
		"connectors":       connectorSummaries,
		"debug":            debugStats,
	}

	s.jsonResponse(w, http.StatusOK, status)
}

// === Connector Management ===

func (s *Server) handleAvailableConnectors(w http.ResponseWriter, r *http.Request) {
	available := s.workbench.GetAvailableConnectors()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"connectors": available,
		"count":      len(available),
	})
}

func (s *Server) handleListConnectors(w http.ResponseWriter, r *http.Request) {
	connectors := s.workbench.ListConnectors()

	result := make([]map[string]interface{}, 0, len(connectors))
	for _, conn := range connectors {
		item := map[string]interface{}{
			"id":           conn.ID,
			"provider":     conn.Provider,
			"name":         conn.Name,
			"connector_id": conn.ConnectorID.String(),
			"installed":    conn.Installed,
			"created_at":   conn.CreatedAt,
		}
		if conn.storage != nil {
			stats := conn.storage.GetStats()
			item["storage"] = map[string]interface{}{
				"accounts_count":          stats.AccountsCount,
				"payments_count":          stats.PaymentsCount,
				"balances_count":          stats.BalancesCount,
				"external_accounts_count": stats.ExternalAccountsCount,
			}
		}
		result = append(result, item)
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"connectors": result,
		"count":      len(result),
	})
}

func (s *Server) handleCreateConnector(w http.ResponseWriter, r *http.Request) {
	var req CreateConnectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	conn, err := s.workbench.CreateConnector(r.Context(), req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":           conn.ID,
		"provider":     conn.Provider,
		"name":         conn.Name,
		"connector_id": conn.ConnectorID.String(),
		"installed":    conn.Installed,
		"created_at":   conn.CreatedAt,
	})
}

func (s *Server) handleDeleteConnector(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil {
		return // Error already sent by middleware
	}

	if err := s.workbench.DeleteConnector(r.Context(), conn.ID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// === Connector Status ===

func (s *Server) handleConnectorStatus(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil {
		return
	}

	var fetchStatus interface{}
	if conn.engine != nil {
		fetchStatus = conn.engine.GetFetchStatus()
	}

	status := map[string]interface{}{
		"id":           conn.ID,
		"provider":     conn.Provider,
		"name":         conn.Name,
		"connector_id": conn.ConnectorID.String(),
		"installed":    conn.Installed,
		"created_at":   conn.CreatedAt,
		"fetch_status": fetchStatus,
	}

	if conn.storage != nil {
		stats := conn.storage.GetStats()
		status["storage"] = map[string]interface{}{
			"accounts_count":          stats.AccountsCount,
			"payments_count":          stats.PaymentsCount,
			"balances_count":          stats.BalancesCount,
			"external_accounts_count": stats.ExternalAccountsCount,
			"last_updated":            stats.LastUpdated,
		}
	}

	s.jsonResponse(w, http.StatusOK, status)
}

// === Install/Uninstall ===

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil {
		return
	}

	if err := s.workbench.InstallConnector(r.Context(), conn.ID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "installed"})
}

func (s *Server) handleUninstall(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil {
		return
	}

	if err := s.workbench.UninstallConnector(r.Context(), conn.ID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "uninstalled"})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil {
		return
	}

	if conn.engine != nil {
		conn.engine.ResetFetchState()
	}
	if conn.storage != nil {
		conn.storage.Clear()
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "reset"})
}

// === Fetch Operations ===

type fetchRequest struct {
	FromPayload  json.RawMessage `json:"from_payload,omitempty"`
	ConnectionID *string         `json:"connection_id,omitempty"`
	InnerPayload json.RawMessage `json:"inner_payload,omitempty"`
}

// resolveFromPayload resolves the from_payload for a fetch request.
// If connection_id is set and from_payload is empty, it builds an
// OpenBankingForwardedUserFromPayload from the stored connection.
func (s *Server) resolveFromPayload(conn *ConnectorInstance, req *fetchRequest) (json.RawMessage, error) {
	if len(req.FromPayload) > 0 {
		return req.FromPayload, nil
	}
	if req.ConnectionID == nil {
		return nil, nil
	}

	obConn, ok := conn.storage.GetOBConnection(*req.ConnectionID)
	if !ok {
		return nil, fmt.Errorf("open banking connection %q not found", *req.ConnectionID)
	}

	return BuildOBFromPayload(obConn, conn.ConnectorID, req.InnerPayload)
}

func (s *Server) handleFetchAccounts(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req fetchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	fromPayload, err := s.resolveFromPayload(conn, &req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := conn.engine.FetchAccountsOnePage(r.Context(), fromPayload)
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
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req fetchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	fromPayload, err := s.resolveFromPayload(conn, &req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := conn.engine.FetchPaymentsOnePage(r.Context(), fromPayload)
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
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req fetchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	fromPayload, err := s.resolveFromPayload(conn, &req)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := conn.engine.FetchBalancesOnePage(r.Context(), fromPayload)
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

func (s *Server) handleFetchExternalAccounts(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req fetchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	resp, err := conn.engine.FetchExternalAccountsOnePage(r.Context(), req.FromPayload)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"external_accounts": resp.ExternalAccounts,
		"has_more":          resp.HasMore,
		"count":             len(resp.ExternalAccounts),
	})
}

func (s *Server) handleFetchAll(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	if err := conn.engine.RunOneCycle(r.Context()); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats := conn.storage.GetStats()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "complete",
		"accounts": stats.AccountsCount,
		"payments": stats.PaymentsCount,
		"balances": stats.BalancesCount,
	})
}

// === Write Operations ===

func (s *Server) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	s.errorResponse(w, http.StatusNotImplemented, "transfer creation not yet implemented in workbench")
}

func (s *Server) handleCreatePayout(w http.ResponseWriter, r *http.Request) {
	s.errorResponse(w, http.StatusNotImplemented, "payout creation not yet implemented in workbench")
}

// === Data Endpoints ===

func (s *Server) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	accounts := conn.storage.GetAccounts()
	response := ToAccountResponses(accounts)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"accounts": response,
		"count":    len(response),
	})
}

func (s *Server) handleGetPayments(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	payments := conn.storage.GetPayments()
	response := ToPaymentResponses(payments)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"payments": response,
		"count":    len(response),
	})
}

func (s *Server) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	accountRef := r.URL.Query().Get("account")
	var response []BalanceResponse

	if accountRef != "" {
		response = ToBalanceResponses(conn.storage.GetBalancesForAccount(accountRef))
	} else {
		response = ToBalanceResponses(conn.storage.GetBalances())
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"balances": response,
	})
}

func (s *Server) handleGetExternalAccounts(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	accounts := conn.storage.GetExternalAccounts()
	response := ToAccountResponses(accounts)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"external_accounts": response,
		"count":             len(response),
	})
}

func (s *Server) handleGetOthers(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	name := r.URL.Query().Get("name")
	var result interface{}

	if name != "" {
		result = conn.storage.GetOthers(name)
	} else {
		result = conn.storage.GetAllOthers()
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"others": result,
	})
}

func (s *Server) handleGetStates(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	states := conn.storage.GetAllStates()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"states": states,
	})
}

func (s *Server) handleGetTasksTree(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	tree := conn.storage.GetTasksTree()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"tasks_tree": tree,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	snapshot := conn.storage.Export()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=workbench-export-%s-%s.json", conn.ID, time.Now().Format("20060102-150405")))
	_ = json.NewEncoder(w).Encode(snapshot)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

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

	conn.storage.Import(snapshot)
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "imported"})
}

// === Debug Endpoints (Global) ===

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
	conn := s.getConnector(r)
	if conn == nil {
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"id":           conn.ID,
		"provider":     conn.Provider,
		"connector_id": conn.ConnectorID.String(),
		"installed":    conn.Installed,
	})
}

func (s *Server) handleSetPageSize(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

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

	conn.engine.SetPageSize(req.PageSize)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"page_size": req.PageSize})
}

// === Introspection Endpoints ===

func (s *Server) handleIntrospectInfo(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.introspector == nil {
		s.errorResponse(w, http.StatusBadRequest, "introspector not available")
		return
	}

	info, err := conn.introspector.GetInfo()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, info)
}

func (s *Server) handleIntrospectFiles(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.introspector == nil {
		s.errorResponse(w, http.StatusBadRequest, "introspector not available")
		return
	}

	files, err := conn.introspector.GetFileTree()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"files": files})
}

func (s *Server) handleIntrospectFile(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.introspector == nil {
		s.errorResponse(w, http.StatusBadRequest, "introspector not available")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		s.errorResponse(w, http.StatusBadRequest, "path parameter required")
		return
	}

	file, err := conn.introspector.GetFile(path)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, file)
}

func (s *Server) handleIntrospectSearch(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.introspector == nil {
		s.errorResponse(w, http.StatusBadRequest, "introspector not available")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.errorResponse(w, http.StatusBadRequest, "q parameter required")
		return
	}

	results, err := conn.introspector.SearchCode(query)
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
	conn := s.getConnector(r)
	if conn == nil || conn.tasks == nil {
		s.errorResponse(w, http.StatusBadRequest, "task tracker not available")
		return
	}

	summary := conn.tasks.GetSummary()
	if conn.engine != nil {
		summary.IsRunning = conn.engine.IsRunning()
	}
	s.jsonResponse(w, http.StatusOK, summary)
}

func (s *Server) handleGetTaskExecutions(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.tasks == nil {
		s.errorResponse(w, http.StatusBadRequest, "task tracker not available")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	executions := conn.tasks.GetExecutions(limit)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}

func (s *Server) handleTaskStep(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.tasks == nil {
		s.errorResponse(w, http.StatusBadRequest, "task tracker not available")
		return
	}

	conn.tasks.Step()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "stepped"})
}

func (s *Server) handleSetStepMode(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.tasks == nil {
		s.errorResponse(w, http.StatusBadRequest, "task tracker not available")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request")
		return
	}

	conn.tasks.SetStepMode(req.Enabled)
	s.jsonResponse(w, http.StatusOK, map[string]bool{"step_mode": req.Enabled})
}

func (s *Server) handleResetTasks(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.tasks == nil {
		s.errorResponse(w, http.StatusBadRequest, "task tracker not available")
		return
	}

	conn.tasks.Reset()
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
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	operation := r.URL.Query().Get("operation")
	tagsParam := r.URL.Query().Get("tags")
	var tags []string
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	list := conn.snapshots.List(operation, tags)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"snapshots": list,
		"count":     len(list),
	})
}

func (s *Server) handleSnapshotStats(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	stats := conn.snapshots.Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	id := chi.URLParam(r, "id")
	snapshot := conn.snapshots.Get(id)
	if snapshot == nil {
		s.errorResponse(w, http.StatusNotFound, "snapshot not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, snapshot)
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	var snapshot Snapshot
	if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := conn.snapshots.Save(&snapshot); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, snapshot)
}

func (s *Server) handleCreateSnapshotFromCapture(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
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

	snapshot, err := conn.snapshots.SaveFromCapture(captureID, req.Name, req.Operation, req.Description, req.Tags)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, snapshot)
}

func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	id := chi.URLParam(r, "id")
	if err := conn.snapshots.Delete(id); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleClearSnapshots(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
		return
	}

	conn.snapshots.Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleExportSnapshots(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
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

	if err := conn.snapshots.ExportToDir(req.Directory); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "exported",
		"directory": req.Directory,
	})
}

func (s *Server) handleImportSnapshots(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "snapshots not available")
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

	count, err := conn.snapshots.ImportFromDir(req.Directory)
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
	conn := s.getConnector(r)
	if conn == nil || conn.testGen == nil {
		s.errorResponse(w, http.StatusBadRequest, "test generator not available")
		return
	}

	result, err := conn.testGen.PreviewTest()
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleGenerateTests(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.testGen == nil || conn.snapshots == nil {
		s.errorResponse(w, http.StatusBadRequest, "test generator not available")
		return
	}

	var req struct {
		OutputDir string `json:"output_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	result, err := conn.testGen.Generate()
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

// === Schema Inference Endpoints ===

func (s *Server) handleListSchemas(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	list := conn.schemas.ListSchemas()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"schemas": list,
		"count":   len(list),
	})
}

func (s *Server) handleSchemaStats(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	stats := conn.schemas.Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	operation := chi.URLParam(r, "operation")
	schema := conn.schemas.GetSchema(operation)
	if schema == nil {
		s.errorResponse(w, http.StatusNotFound, "schema not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, schema)
}

func (s *Server) handleInferSchema(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
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

	schema, err := conn.schemas.InferFromJSON(req.Operation, req.Endpoint, req.Method, req.Data)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, schema)
}

func (s *Server) handleSaveSchemaBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	var req struct {
		Operation string `json:"operation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := conn.schemas.SaveBaseline(req.Operation); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Server) handleSaveAllSchemaBaselines(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	count := conn.schemas.SaveAllBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "saved",
		"count":  count,
	})
}

func (s *Server) handleListSchemaBaselines(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	list := conn.schemas.ListBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"baselines": list,
		"count":     len(list),
	})
}

func (s *Server) handleCompareSchema(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	operation := chi.URLParam(r, "operation")
	diff, err := conn.schemas.CompareWithBaseline(operation)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, diff)
}

func (s *Server) handleClearSchemas(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.schemas == nil {
		s.errorResponse(w, http.StatusBadRequest, "schema manager not available")
		return
	}

	conn.schemas.Clear()
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// === Data Baseline Endpoints ===

func (s *Server) handleListBaselines(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	list := conn.baselines.ListBaselines()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"baselines": list,
		"count":     len(list),
	})
}

func (s *Server) handleGetBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	baseline := conn.baselines.GetBaseline(id)
	if baseline == nil {
		s.errorResponse(w, http.StatusNotFound, "baseline not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, baseline)
}

func (s *Server) handleSaveBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
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

	baseline, err := conn.baselines.SaveBaseline(req.Name)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, baseline)
}

func (s *Server) handleDeleteBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	if err := conn.baselines.DeleteBaseline(id); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleCompareBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	diff, err := conn.baselines.CompareWithCurrent(id)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, diff)
}

func (s *Server) handleExportBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	id := chi.URLParam(r, "id")
	data, err := conn.baselines.Export(id)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", id))
	_, _ = w.Write(data)
}

func (s *Server) handleImportBaseline(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.baselines == nil {
		s.errorResponse(w, http.StatusBadRequest, "baseline manager not available")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "failed to read body")
		return
	}

	baseline, err := conn.baselines.Import(data)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, baseline)
}

// === Open Banking Endpoints ===

func (s *Server) handleOBCreateUser(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	resp, psuID, err := conn.engine.CreateUser(r.Context())
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string]interface{}{
		"psu_id": psuID.String(),
	}
	if resp.PSPUserID != nil {
		result["psp_user_id"] = *resp.PSPUserID
	}
	if resp.PermanentToken != nil {
		result["permanent_token"] = resp.PermanentToken
	}
	if resp.Metadata != nil {
		result["metadata"] = resp.Metadata
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleOBCreateUserLink(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req struct {
		PSUID string `json:"psu_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	psuID, err := uuid.Parse(req.PSUID)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid psu_id: "+err.Error())
		return
	}

	resp, err := conn.engine.CreateUserLink(r.Context(), psuID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string]interface{}{
		"link": resp.Link,
	}
	if resp.TemporaryLinkToken != nil {
		result["temporary_token"] = resp.TemporaryLinkToken
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleOBCompleteUserLink(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.engine == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	var req struct {
		PSUID       string              `json:"psu_id"`
		QueryValues map[string][]string `json:"query_values,omitempty"`
		Headers     map[string][]string `json:"headers,omitempty"`
		Body        string              `json:"body,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	psuID, err := uuid.Parse(req.PSUID)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid psu_id: "+err.Error())
		return
	}

	resp, err := conn.engine.CompleteUserLink(r.Context(), psuID, req.QueryValues, req.Headers, []byte(req.Body))
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if resp.Error != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"error": resp.Error.Error,
		})
		return
	}

	if resp.Success != nil {
		// Store connections
		var stored []StoredOBConnection
		for _, pspConn := range resp.Success.Connections {
			sc := StoredOBConnection{
				ConnectionID: pspConn.ConnectionID,
				ConnectorID:  conn.ConnectorID,
				PSUID:        psuID,
				AccessToken:  pspConn.AccessToken,
				Metadata:     pspConn.Metadata,
				CreatedAt:    pspConn.CreatedAt,
			}
			conn.storage.StoreOBConnection(sc)
			stored = append(stored, sc)
		}

		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"connections": stored,
			"count":       len(stored),
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"connections": []StoredOBConnection{},
		"count":       0,
	})
}

func (s *Server) handleOBListConnections(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	conns := conn.storage.GetOBConnections()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"connections": conns,
		"count":       len(conns),
	})
}

func (s *Server) handleOBDeleteConnection(w http.ResponseWriter, r *http.Request) {
	conn := s.getConnector(r)
	if conn == nil || conn.storage == nil {
		s.errorResponse(w, http.StatusBadRequest, "connector not ready")
		return
	}

	connectionID := chi.URLParam(r, "connectionID")
	if !conn.storage.DeleteOBConnection(connectionID) {
		s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("connection %s not found", connectionID))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
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
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; padding: 40px; }
        h1 { color: #58a6ff; margin-bottom: 20px; }
        p { margin-bottom: 10px; color: #8b949e; }
        a { color: #58a6ff; }
        code { background: #161b22; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Connector Workbench (Multi-Connector)</h1>
    <p>The full UI is not embedded. Build it with:</p>
    <p><code>cd tools/workbench/ui && npm install && npm run build</code></p>
    <p>API available at <a href="/api/status">/api/status</a></p>
    <p>Available connectors at <a href="/api/connectors/available">/api/connectors/available</a></p>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(html))
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

// === Generic Connector Server ===
// These handlers expose the standard generic connector API so remote services
// can connect to locally-running connectors for integration testing.

// genericServerMiddleware validates API key for the generic server.
// Handlers get the connector directly from the workbench for clearer data flow.
func (s *Server) genericServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, apiKey := s.workbench.GetGenericServerConnector()
		if conn == nil {
			s.genericError(w, http.StatusServiceUnavailable, "Generic server not configured", "No connector is configured for the generic server. Use the API to set one.")
			return
		}

		if !conn.Installed {
			s.genericError(w, http.StatusServiceUnavailable, "Connector not installed", "The configured connector is not installed.")
			return
		}

		// Validate API key if configured
		if apiKey != "" {
			authHeader := r.Header.Get("Authorization")
			providedKey := ""
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// Also check X-API-Key header
				providedKey = r.Header.Get("X-API-Key")
			}
			if providedKey != apiKey {
				s.genericError(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing API key")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// getGenericConnector returns the configured generic connector directly from the workbench.
// This avoids storing connector references in context, providing clearer data flow.
func (s *Server) getGenericConnector() *ConnectorInstance {
	conn, _ := s.workbench.GetGenericServerConnector()
	return conn
}

func (s *Server) genericError(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"Title":  title,
		"Detail": detail,
	})
}

// handleGenericServerStatus returns the generic server configuration.
func (s *Server) handleGenericServerStatus(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, s.workbench.GetGenericServerStatus())
}

// handleSetGenericConnector sets the connector for the generic server.
func (s *Server) handleSetGenericConnector(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnectorID string `json:"connector_id"`
		APIKey      string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := s.workbench.SetGenericServerConnector(req.ConnectorID, req.APIKey); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, s.workbench.GetGenericServerStatus())
}

// handleGenericAccounts handles GET /generic/accounts
func (s *Server) handleGenericAccounts(w http.ResponseWriter, r *http.Request) {
	conn := s.getGenericConnector()
	if conn == nil {
		return // Error already sent by middleware
	}

	// Parse query params
	createdAtFrom := r.URL.Query().Get("createdAtFrom")

	// Return cached accounts from storage
	storedAccounts := conn.storage.GetAccounts()

	// Transform to generic format
	accounts := make([]map[string]interface{}, 0, len(storedAccounts))
	for _, acc := range storedAccounts {
		// Filter by createdAtFrom if specified
		if createdAtFrom != "" {
			if fromTime, err := time.Parse(time.RFC3339, createdAtFrom); err == nil {
				if acc.CreatedAt.Before(fromTime) {
					continue
				}
			}
		}

		account := map[string]interface{}{
			"id":        acc.Reference,
			"createdAt": acc.CreatedAt.Format(time.RFC3339),
		}
		if acc.Name != nil {
			account["accountName"] = *acc.Name
		} else {
			account["accountName"] = acc.Reference
		}
		if len(acc.Metadata) > 0 {
			account["metadata"] = acc.Metadata
		}
		accounts = append(accounts, account)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(accounts)
}

// handleGenericBalances handles GET /generic/accounts/{accountId}/balances
func (s *Server) handleGenericBalances(w http.ResponseWriter, r *http.Request) {
	conn := s.getGenericConnector()
	if conn == nil {
		return
	}

	accountID := chi.URLParam(r, "accountId")
	if accountID == "" {
		s.genericError(w, http.StatusBadRequest, "Missing account ID", "accountId path parameter is required")
		return
	}

	// Return cached balances from storage
	storedBalances := conn.storage.GetBalancesForAccount(accountID)

	// Transform to generic format
	balances := make([]map[string]interface{}, 0, len(storedBalances))
	var latestTime time.Time
	for _, bal := range storedBalances {
		balances = append(balances, map[string]interface{}{
			"amount":   bal.Amount.String(),
			"currency": bal.Asset,
		})
		if bal.CreatedAt.After(latestTime) {
			latestTime = bal.CreatedAt
		}
	}

	// Use latest balance time, or now if no balances
	if latestTime.IsZero() {
		latestTime = time.Now()
	}

	result := map[string]interface{}{
		"id":        accountID + "-balance",
		"accountID": accountID,
		"at":        latestTime.Format(time.RFC3339),
		"balances":  balances,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleGenericBeneficiaries handles GET /generic/beneficiaries
func (s *Server) handleGenericBeneficiaries(w http.ResponseWriter, r *http.Request) {
	conn := s.getGenericConnector()
	if conn == nil {
		return
	}

	// Parse query params
	createdAtFrom := r.URL.Query().Get("createdAtFrom")

	// Return cached external accounts from storage
	storedExternalAccounts := conn.storage.GetExternalAccounts()

	// Transform to generic format
	beneficiaries := make([]map[string]interface{}, 0, len(storedExternalAccounts))
	for _, ext := range storedExternalAccounts {
		if createdAtFrom != "" {
			if fromTime, err := time.Parse(time.RFC3339, createdAtFrom); err == nil {
				if ext.CreatedAt.Before(fromTime) {
					continue
				}
			}
		}

		beneficiary := map[string]interface{}{
			"id":        ext.Reference,
			"createdAt": ext.CreatedAt.Format(time.RFC3339),
		}
		if ext.Name != nil {
			beneficiary["ownerName"] = *ext.Name
		} else {
			beneficiary["ownerName"] = ext.Reference
		}
		if len(ext.Metadata) > 0 {
			beneficiary["metadata"] = ext.Metadata
		}
		beneficiaries = append(beneficiaries, beneficiary)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(beneficiaries)
}

// handleGenericTransactions handles GET /generic/transactions
func (s *Server) handleGenericTransactions(w http.ResponseWriter, r *http.Request) {
	conn := s.getGenericConnector()
	if conn == nil {
		return
	}

	// Parse query params
	updatedAtFrom := r.URL.Query().Get("updatedAtFrom")

	// Return cached payments from storage
	storedPayments := conn.storage.GetPayments()

	// Transform to generic format
	transactions := make([]map[string]interface{}, 0, len(storedPayments))
	for _, pmt := range storedPayments {
		if updatedAtFrom != "" {
			if fromTime, err := time.Parse(time.RFC3339, updatedAtFrom); err == nil {
				if pmt.CreatedAt.Before(fromTime) {
					continue
				}
			}
		}

		tx := map[string]interface{}{
			"id":        pmt.Reference,
			"createdAt": pmt.CreatedAt.Format(time.RFC3339),
			"updatedAt": pmt.CreatedAt.Format(time.RFC3339), // Use createdAt as updatedAt if not available
			"currency":  pmt.Asset,
			"amount":    pmt.Amount.String(),
			"type":      mapPaymentTypeToGeneric(pmt.Type),
			"status":    mapPaymentStatusToGeneric(pmt.Status),
		}

		if pmt.Scheme != models.PAYMENT_SCHEME_UNKNOWN {
			tx["scheme"] = pmt.Scheme.String()
		}
		if pmt.SourceAccountReference != nil {
			tx["sourceAccountID"] = *pmt.SourceAccountReference
		}
		if pmt.DestinationAccountReference != nil {
			tx["destinationAccountID"] = *pmt.DestinationAccountReference
		}
		if len(pmt.Metadata) > 0 {
			tx["metadata"] = pmt.Metadata
		}

		transactions = append(transactions, tx)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transactions)
}

func mapPaymentTypeToGeneric(t models.PaymentType) string {
	switch t {
	case models.PAYMENT_TYPE_PAYIN:
		return "PAYIN"
	case models.PAYMENT_TYPE_PAYOUT:
		return "PAYOUT"
	case models.PAYMENT_TYPE_TRANSFER:
		return "TRANSFER"
	default:
		return "TRANSFER"
	}
}

func mapPaymentStatusToGeneric(s models.PaymentStatus) string {
	switch s {
	case models.PAYMENT_STATUS_PENDING:
		return "PENDING"
	case models.PAYMENT_STATUS_SUCCEEDED:
		return "SUCCEEDED"
	case models.PAYMENT_STATUS_FAILED, models.PAYMENT_STATUS_CANCELLED:
		return "FAILED"
	default:
		return "PENDING"
	}
}
