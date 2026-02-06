// Package workbench provides a lightweight development environment for building
// and testing payment connectors without requiring Temporal, PostgreSQL, or other
// heavy infrastructure.
//
// The workbench is designed to:
//   - Run entirely in-memory with optional SQLite persistence
//   - Provide step-by-step execution of connector operations
//   - Offer comprehensive debugging and inspection capabilities
//   - Expose an HTTP API and Web UI for interactive development
//   - Support multiple connectors simultaneously
package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

// Global debug transport for HTTP interception
var globalDebugTransport *DebugTransport

// Config holds the workbench configuration.
type Config struct {
	// ListenAddr is the HTTP server listen address
	ListenAddr string
	// EnableUI enables the embedded web UI
	EnableUI bool
	// PersistPath is the optional path to persist state (empty = in-memory only)
	PersistPath string
	// DebugMode enables debug features like dummypay connector
	DebugMode bool
	// DefaultPageSize is the default page size for fetch operations
	DefaultPageSize int
}

// DefaultConfig returns a default workbench configuration.
func DefaultConfig() Config {
	return Config{
		ListenAddr:      "127.0.0.1:8080",
		EnableUI:        true,
		DebugMode:       true,
		DefaultPageSize: 25,
	}
}

// ConnectorInstance represents a single connector instance in the workbench.
type ConnectorInstance struct {
	ID           string                 `json:"id"`
	Provider     string                 `json:"provider"`
	Name         string                 `json:"name"`
	ConnectorID  models.ConnectorID     `json:"connector_id"`
	Config       json.RawMessage        `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
	Installed    bool                   `json:"installed"`

	// Internal components (not serialized)
	plugin       models.Plugin
	engine       *Engine
	storage      *MemoryStorage
	tasks        *TaskTracker
	introspector *Introspector
	snapshots    *SnapshotManager
	testGen      *TestGenerator
	schemas      *SchemaManager
	baselines    *BaselineManager
}

// Workbench is the main connector development environment.
type Workbench struct {
	config Config
	logger logging.Logger

	// Shared components
	debug     *DebugStore
	server    *Server
	transport *DebugTransport
	replayer  *Replayer

	// Original transport to restore on shutdown
	originalTransport http.RoundTripper

	// Multi-connector management
	connectors map[string]*ConnectorInstance
	connMu     sync.RWMutex

	// Generic server configuration (for remote integration testing)
	genericConnectorID string
	genericAPIKey      string

	// Lifecycle
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// New creates a new Workbench instance.
func New(cfg Config, logger logging.Logger) (*Workbench, error) {
	debug := NewDebugStore(1000) // Keep last 1000 debug entries

	w := &Workbench{
		config:     cfg,
		logger:     logger,
		debug:      debug,
		replayer:   NewReplayer(debug),
		connectors: make(map[string]*ConnectorInstance),
		stopChan:   make(chan struct{}),
	}

	// Install HTTP debug transport
	w.transport, w.originalTransport = InstallGlobalTransport(w.debug)
	globalDebugTransport = w.transport
	logger.Info("HTTP debug transport installed - all outbound HTTP traffic will be captured")

	// Create HTTP server
	w.server = NewServer(w, cfg.ListenAddr, cfg.EnableUI)

	return w, nil
}

// Start starts the workbench.
func (w *Workbench) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("workbench already running")
	}
	w.running = true
	w.mu.Unlock()

	w.logger.Info("Starting connector workbench (multi-connector mode)")
	w.logger.Infof("  HTTP API: http://%s", w.config.ListenAddr)
	if w.config.EnableUI {
		w.logger.Infof("  Web UI: http://%s/ui", w.config.ListenAddr)
	}

	// Start HTTP server
	go func() {
		if err := w.server.Start(); err != nil && err != http.ErrServerClosed {
			w.logger.Errorf("HTTP server error: %v", err)
		}
	}()

	w.logger.Info("Workbench ready!")
	w.printUsage()

	return nil
}

// Stop stops the workbench.
func (w *Workbench) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	close(w.stopChan)
	w.mu.Unlock()

	w.logger.Info("Stopping workbench...")

	// Stop HTTP server
	if err := w.server.Stop(ctx); err != nil {
		w.logger.Errorf("Error stopping HTTP server: %v", err)
	}

	// Uninstall all connectors
	w.connMu.Lock()
	for id, conn := range w.connectors {
		if conn.Installed && conn.engine != nil {
			if err := conn.engine.Uninstall(ctx); err != nil {
				w.logger.Errorf("Error uninstalling connector %s: %v", id, err)
			}
		}
	}
	w.connMu.Unlock()

	// Restore original HTTP transport
	if w.originalTransport != nil {
		RestoreGlobalTransport(w.originalTransport)
		w.logger.Info("HTTP debug transport removed")
	}

	return nil
}

// Wait blocks until the workbench is stopped.
func (w *Workbench) Wait() {
	<-w.stopChan
}

// CreateConnectorRequest is the request to create a new connector instance.
type CreateConnectorRequest struct {
	Provider string          `json:"provider"`
	Name     string          `json:"name"`
	Config   json.RawMessage `json:"config"`
}

// CreateConnector creates a new connector instance.
func (w *Workbench) CreateConnector(ctx context.Context, req CreateConnectorRequest) (*ConnectorInstance, error) {
	if req.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if len(req.Config) == 0 {
		return nil, fmt.Errorf("config is required")
	}

	// Generate instance ID
	instanceID := uuid.New().String()[:8]
	if req.Name != "" {
		instanceID = req.Name
	}

	// Create connector ID
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  req.Provider,
	}

	// Load the plugin
	plugin, err := registry.GetPlugin(
		connectorID,
		w.logger,
		req.Provider,
		fmt.Sprintf("workbench-%s-%s", req.Provider, instanceID),
		req.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %q: %w", req.Provider, err)
	}

	// Create instance-specific components
	storage := NewMemoryStorage()
	tasks := NewTaskTracker()
	snapshots := NewSnapshotManager(req.Provider, w.debug)
	schemas := NewSchemaManager(req.Provider)

	// Link schemas to transport for auto-inference
	w.transport.Schemas = schemas

	instance := &ConnectorInstance{
		ID:           instanceID,
		Provider:     req.Provider,
		Name:         req.Name,
		ConnectorID:  connectorID,
		Config:       req.Config,
		CreatedAt:    time.Now(),
		Installed:    false,
		plugin:       plugin,
		storage:      storage,
		tasks:        tasks,
		introspector: NewIntrospector(req.Provider, connectorID),
		snapshots:    snapshots,
		testGen:      NewTestGenerator(snapshots, req.Provider),
		schemas:      schemas,
		baselines:    NewBaselineManager(req.Provider, storage),
	}

	// Create the engine
	instance.engine = NewEngine(connectorID, plugin, storage, w.debug, tasks, w.logger)

	// Set default page size
	if w.config.DefaultPageSize > 0 {
		instance.engine.SetPageSize(w.config.DefaultPageSize)
	}

	// Store the instance
	w.connMu.Lock()
	w.connectors[instanceID] = instance
	w.connMu.Unlock()

	w.logger.Infof("Created connector instance %s (provider: %s)", instanceID, req.Provider)

	return instance, nil
}

// InstallConnector installs a connector instance.
func (w *Workbench) InstallConnector(ctx context.Context, instanceID string) error {
	instance := w.GetConnector(instanceID)
	if instance == nil {
		return fmt.Errorf("connector %s not found", instanceID)
	}

	if instance.Installed {
		return fmt.Errorf("connector %s is already installed", instanceID)
	}

	if err := instance.engine.Install(ctx); err != nil {
		return fmt.Errorf("failed to install connector: %w", err)
	}

	w.connMu.Lock()
	instance.Installed = true
	w.connMu.Unlock()

	w.logger.Infof("Installed connector %s", instanceID)
	return nil
}

// UninstallConnector uninstalls a connector instance.
func (w *Workbench) UninstallConnector(ctx context.Context, instanceID string) error {
	instance := w.GetConnector(instanceID)
	if instance == nil {
		return fmt.Errorf("connector %s not found", instanceID)
	}

	if !instance.Installed {
		return fmt.Errorf("connector %s is not installed", instanceID)
	}

	if err := instance.engine.Uninstall(ctx); err != nil {
		return fmt.Errorf("failed to uninstall connector: %w", err)
	}

	w.connMu.Lock()
	instance.Installed = false
	w.connMu.Unlock()

	w.logger.Infof("Uninstalled connector %s", instanceID)
	return nil
}

// DeleteConnector removes a connector instance.
func (w *Workbench) DeleteConnector(ctx context.Context, instanceID string) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()

	instance, ok := w.connectors[instanceID]
	if !ok {
		return fmt.Errorf("connector %s not found", instanceID)
	}

	// Uninstall if installed
	if instance.Installed && instance.engine != nil {
		if err := instance.engine.Uninstall(ctx); err != nil {
			w.logger.Errorf("Error uninstalling connector %s during delete: %v", instanceID, err)
		}
	}

	delete(w.connectors, instanceID)
	w.logger.Infof("Deleted connector %s", instanceID)
	return nil
}

// GetConnector returns a connector instance by ID.
func (w *Workbench) GetConnector(instanceID string) *ConnectorInstance {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	return w.connectors[instanceID]
}

// ListConnectors returns all connector instances.
func (w *Workbench) ListConnectors() []*ConnectorInstance {
	w.connMu.RLock()
	defer w.connMu.RUnlock()

	result := make([]*ConnectorInstance, 0, len(w.connectors))
	for _, conn := range w.connectors {
		result = append(result, conn)
	}
	return result
}

// SetGenericServerConnector sets the connector to use for the generic server API.
func (w *Workbench) SetGenericServerConnector(connectorID string, apiKey string) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()

	if connectorID != "" {
		if _, ok := w.connectors[connectorID]; !ok {
			return fmt.Errorf("connector %s not found", connectorID)
		}
	}

	w.genericConnectorID = connectorID
	w.genericAPIKey = apiKey
	return nil
}

// GetGenericServerConnector returns the connector configured for the generic server.
func (w *Workbench) GetGenericServerConnector() (*ConnectorInstance, string) {
	w.connMu.RLock()
	defer w.connMu.RUnlock()

	if w.genericConnectorID == "" {
		return nil, ""
	}
	return w.connectors[w.genericConnectorID], w.genericAPIKey
}

// GetGenericServerStatus returns the generic server configuration status.
func (w *Workbench) GetGenericServerStatus() map[string]interface{} {
	w.connMu.RLock()
	defer w.connMu.RUnlock()

	status := map[string]interface{}{
		"enabled":      w.genericConnectorID != "",
		"connector_id": w.genericConnectorID,
		"has_api_key":  w.genericAPIKey != "",
		"endpoint":     fmt.Sprintf("http://%s/generic", w.config.ListenAddr),
	}

	if w.genericConnectorID != "" {
		if conn, ok := w.connectors[w.genericConnectorID]; ok {
			status["connector_provider"] = conn.Provider
			status["connector_installed"] = conn.Installed
		}
	}

	return status
}

// AvailableConnector represents an available connector type.
type AvailableConnector struct {
	Provider   string             `json:"provider"`
	Config     registry.Config    `json:"config"`
	PluginType models.PluginType  `json:"plugin_type"`
}

// GetAvailableConnectors returns all available connector types from the registry.
func (w *Workbench) GetAvailableConnectors() []AvailableConnector {
	configs := registry.GetConfigs(w.config.DebugMode)
	result := make([]AvailableConnector, 0, len(configs))

	for provider, config := range configs {
		pluginType, _ := registry.GetPluginType(provider)
		result = append(result, AvailableConnector{
			Provider:   provider,
			Config:     config,
			PluginType: pluginType,
		})
	}

	// Sort by provider name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Provider < result[j].Provider
	})

	return result
}

func (w *Workbench) printUsage() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              CONNECTOR WORKBENCH (Multi-Connector)            ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║ HTTP API Endpoints:                                           ║")
	fmt.Println("║   GET  /api/status                    - Workbench status      ║")
	fmt.Println("║   GET  /api/connectors/available      - List available types  ║")
	fmt.Println("║   GET  /api/connectors                - List instances        ║")
	fmt.Println("║   POST /api/connectors                - Create instance       ║")
	fmt.Println("║   DELETE /api/connectors/{id}         - Delete instance       ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║ Connector-specific (replace {id} with instance ID):           ║")
	fmt.Println("║   POST /api/connectors/{id}/install   - Install connector     ║")
	fmt.Println("║   POST /api/connectors/{id}/uninstall - Uninstall connector   ║")
	fmt.Println("║   POST /api/connectors/{id}/fetch/... - Fetch operations      ║")
	fmt.Println("║   GET  /api/connectors/{id}/data/...  - View data             ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// Debug returns the debug store for direct access.
func (w *Workbench) Debug() *DebugStore {
	return w.debug
}

// Config returns the workbench configuration.
func (w *Workbench) Config() Config {
	return w.config
}

// Transport returns the debug HTTP transport.
func (w *Workbench) Transport() *DebugTransport {
	return w.transport
}

// Replayer returns the HTTP request replayer.
func (w *Workbench) Replayer() *Replayer {
	return w.replayer
}
