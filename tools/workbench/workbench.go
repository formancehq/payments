// Package workbench provides a lightweight development environment for building
// and testing payment connectors without requiring Temporal, PostgreSQL, or other
// heavy infrastructure.
//
// The workbench is designed to:
//   - Run entirely in-memory with optional SQLite persistence
//   - Provide step-by-step execution of connector operations
//   - Offer comprehensive debugging and inspection capabilities
//   - Expose an HTTP API and Web UI for interactive development
package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	// Provider is the connector provider name (e.g., "stripe", "wise")
	Provider string
	// ConnectorConfig is the raw JSON configuration for the connector
	ConnectorConfig json.RawMessage
	// ListenAddr is the HTTP server listen address
	ListenAddr string
	// EnableUI enables the embedded web UI
	EnableUI bool
	// PersistPath is the optional path to persist state (empty = in-memory only)
	PersistPath string
	// AutoPoll enables automatic polling (vs manual step-by-step)
	AutoPoll bool
	// PollInterval is the interval between auto-poll cycles
	PollInterval time.Duration
}

// DefaultConfig returns a default workbench configuration.
func DefaultConfig() Config {
	return Config{
		ListenAddr:   "127.0.0.1:8080",
		EnableUI:     true,
		AutoPoll:     false,
		PollInterval: 30 * time.Second,
	}
}

// Workbench is the main connector development environment.
type Workbench struct {
	config Config
	logger logging.Logger

	// Core components
	storage      *MemoryStorage
	engine       *Engine
	debug        *DebugStore
	server       *Server
	transport    *DebugTransport
	introspector *Introspector
	tasks        *TaskTracker
	replayer     *Replayer
	snapshots    *SnapshotManager
	testGen      *TestGenerator
	schemas      *SchemaManager
	baselines    *BaselineManager

	// Original transport to restore on shutdown
	originalTransport http.RoundTripper

	// Plugin management
	connectorID models.ConnectorID
	plugin      models.Plugin

	// Lifecycle
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// New creates a new Workbench instance.
func New(cfg Config, logger logging.Logger) (*Workbench, error) {
	if cfg.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if len(cfg.ConnectorConfig) == 0 {
		return nil, fmt.Errorf("connector config is required")
	}

	debug := NewDebugStore(1000) // Keep last 1000 debug entries
	storage := NewMemoryStorage()
	snapshots := NewSnapshotManager(cfg.Provider, debug)
	w := &Workbench{
		config:    cfg,
		logger:    logger,
		storage:   storage,
		debug:     debug,
		tasks:     NewTaskTracker(),
		replayer:  NewReplayer(debug),
		snapshots: snapshots,
		testGen:   NewTestGenerator(snapshots, cfg.Provider),
		schemas:   NewSchemaManager(cfg.Provider),
		baselines: NewBaselineManager(cfg.Provider, storage),
		stopChan:  make(chan struct{}),
	}

	// Install HTTP debug transport BEFORE loading the plugin
	// This ensures all HTTP clients created by the plugin will use our transport
	w.transport, w.originalTransport = InstallGlobalTransport(w.debug)
	w.transport.Schemas = w.schemas // Enable auto schema inference
	globalDebugTransport = w.transport
	logger.Info("HTTP debug transport installed - all outbound HTTP traffic will be captured")

	// Create connector ID
	w.connectorID = models.ConnectorID{
		Reference: uuid.New(),
		Provider:  cfg.Provider,
	}

	// Load the plugin
	plugin, err := registry.GetPlugin(
		w.connectorID,
		logger,
		cfg.Provider,
		fmt.Sprintf("workbench-%s", cfg.Provider),
		cfg.ConnectorConfig,
	)
	if err != nil {
		// Restore original transport on error
		RestoreGlobalTransport(w.originalTransport)
		return nil, fmt.Errorf("failed to load plugin %q: %w", cfg.Provider, err)
	}
	w.plugin = plugin

	// Create the dev engine
	w.engine = NewEngine(w.connectorID, plugin, w.storage, w.debug, w.tasks, logger)

	// Create introspector for code analysis
	w.introspector = NewIntrospector(cfg.Provider, w.connectorID)

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

	w.logger.Infof("Starting connector workbench for %s", w.config.Provider)
	w.logger.Infof("  Connector ID: %s", w.connectorID.String())
	w.logger.Infof("  HTTP API: http://%s", w.config.ListenAddr)
	if w.config.EnableUI {
		w.logger.Infof("  Web UI: http://%s/ui", w.config.ListenAddr)
	}

	// Install the connector
	w.logger.Info("Installing connector...")
	if err := w.engine.Install(ctx); err != nil {
		return fmt.Errorf("failed to install connector: %w", err)
	}
	w.logger.Info("Connector installed successfully")

	// Start HTTP server
	go func() {
		if err := w.server.Start(); err != nil && err != http.ErrServerClosed {
			w.logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// Start auto-poll if enabled
	if w.config.AutoPoll {
		go w.autoPollLoop(ctx)
	}

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

	// Uninstall connector
	if err := w.engine.Uninstall(ctx); err != nil {
		w.logger.Errorf("Error uninstalling connector: %v", err)
	}

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

func (w *Workbench) autoPollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.logger.Debug("Auto-poll triggered")
			if err := w.engine.RunOneCycle(ctx); err != nil {
				w.logger.Errorf("Auto-poll error: %v", err)
			}
		}
	}
}

func (w *Workbench) printUsage() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   CONNECTOR WORKBENCH                         ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║ HTTP API Endpoints:                                           ║")
	fmt.Println("║   GET  /status              - Workbench status                ║")
	fmt.Println("║   POST /install             - Install connector               ║")
	fmt.Println("║   POST /uninstall           - Uninstall connector             ║")
	fmt.Println("║   POST /fetch/accounts      - Fetch accounts (one page)       ║")
	fmt.Println("║   POST /fetch/balances      - Fetch balances                  ║")
	fmt.Println("║   POST /fetch/payments      - Fetch payments (one page)       ║")
	fmt.Println("║   POST /fetch/all           - Run full fetch cycle            ║")
	fmt.Println("║   POST /transfer            - Create a transfer               ║")
	fmt.Println("║   POST /payout              - Create a payout                 ║")
	fmt.Println("║   GET  /data/accounts       - List fetched accounts           ║")
	fmt.Println("║   GET  /data/payments       - List fetched payments           ║")
	fmt.Println("║   GET  /data/balances       - List fetched balances           ║")
	fmt.Println("║   GET  /debug/logs          - View debug logs                 ║")
	fmt.Println("║   GET  /debug/state         - View connector state            ║")
	fmt.Println("║   GET  /debug/requests      - View HTTP requests to PSP       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// Engine returns the workbench engine for direct access.
func (w *Workbench) Engine() *Engine {
	return w.engine
}

// Storage returns the workbench storage for direct access.
func (w *Workbench) Storage() *MemoryStorage {
	return w.storage
}

// Debug returns the debug store for direct access.
func (w *Workbench) Debug() *DebugStore {
	return w.debug
}

// ConnectorID returns the connector ID.
func (w *Workbench) ConnectorID() models.ConnectorID {
	return w.connectorID
}

// Plugin returns the loaded plugin.
func (w *Workbench) Plugin() models.Plugin {
	return w.plugin
}

// Config returns the workbench configuration.
func (w *Workbench) Config() Config {
	return w.config
}

// Transport returns the debug HTTP transport.
func (w *Workbench) Transport() *DebugTransport {
	return w.transport
}

// Introspector returns the code introspector.
func (w *Workbench) Introspector() *Introspector {
	return w.introspector
}

// Tasks returns the task tracker.
func (w *Workbench) Tasks() *TaskTracker {
	return w.tasks
}

// Replayer returns the HTTP request replayer.
func (w *Workbench) Replayer() *Replayer {
	return w.replayer
}

// Snapshots returns the snapshot manager.
func (w *Workbench) Snapshots() *SnapshotManager {
	return w.snapshots
}

// TestGenerator returns the test generator.
func (w *Workbench) TestGenerator() *TestGenerator {
	return w.testGen
}

// Schemas returns the schema manager.
func (w *Workbench) Schemas() *SchemaManager {
	return w.schemas
}

// Baselines returns the baseline manager.
func (w *Workbench) Baselines() *BaselineManager {
	return w.baselines
}
