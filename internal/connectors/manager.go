package connectors

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	pluginserrors "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

var (
	ErrNotFound         = errors.New("plugin not loaded in manager")
	ErrValidation       = errors.New("validation error")
	ErrPollingPeriod    = errors.New("polling period invalid")
	ErrInvalidOperation = errors.New("invalid operation")
)

//go:generate mockgen -source manager.go -destination manager_generated.go -package connectors . Manager
type Manager interface {
	Load(models.Connector, bool, bool) (string, json.RawMessage, error)
	Unload(models.ConnectorID)
	GetConfig(models.ConnectorID) (models.Config, error)
	Get(models.ConnectorID) (models.Plugin, error)
}

// Will start, hold, manage and stop Connectors
type manager struct {
	logger logging.Logger

	connectors map[string]connector
	rwMutex    sync.RWMutex

	configurer *Configurer

	debug bool
}

type connector struct {
	plugin models.Plugin
	config models.Config

	validatedConfigJson json.RawMessage
}

func NewManager(
	logger logging.Logger,
	debug bool,
	pollingPeriodDefault time.Duration,
	pollingPeriodMinimum time.Duration,
) *manager {
	configurer, err := NewConfigurer(pollingPeriodDefault, pollingPeriodMinimum)
	if err != nil {
		// NewManager is only expected to be called in modules - we'd rather fail starting the app than
		// start it misconfigured
		log.Panicf("invalid connector polling period configuration: %v", err)
	}
	return &manager{
		logger:     logger,
		configurer: configurer,
		connectors: make(map[string]connector),
		debug:      debug,
	}
}

func (m *manager) Load(connectorModel models.Connector, updateExisting bool, strictValidation bool) (configName string, validatedConfigJson json.RawMessage, err error) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	config := m.configurer.DefaultConfig()
	if err := json.Unmarshal(connectorModel.Config, &config); err != nil {
		return "", nil, err
	}

	// Check if plugin is already installed
	_, ok := m.connectors[connectorModel.ID.String()]
	if ok && !updateExisting {
		return config.Name, m.connectors[connectorModel.ID.String()].validatedConfigJson, nil
	}

	if err := m.configurer.Validate(config); err != nil {
		if !errors.Is(err, ErrPollingPeriod) {
			return "", nil, err
		}

		// strict validation takes place on install/update but not when launching a new instance of the app
		// which is only loading a presumably already validated value from the DB
		if strictValidation {
			return "", nil, fmt.Errorf("%w: %w", models.ErrInvalidConfig, err)
		}
		// if the polling period is lower that the current system default we should still load the plugin
		// since creating validation errors will not change the schedule in temporal
		m.logger.Errorf("connector %q has a low polling period of %s", connectorModel.ID.String(), config.PollingPeriod)
	}

	plugin, err := registry.GetPlugin(connectorModel.ID, m.logger, connectorModel.Provider, config.Name, connectorModel.Config)
	switch {
	case errors.Is(err, pluginserrors.ErrNotImplemented),
		errors.Is(err, pluginserrors.ErrInvalidClientRequest):
		return "", nil, fmt.Errorf("%w: %w", err, ErrValidation)
	case err != nil:
		return "", nil, err
	}

	b, err := combineConfigs(config, plugin.Config())
	if err != nil {
		return "", nil, fmt.Errorf("failed to combine configs: %w", err)
	}

	plugin.ScheduleForDeletion(connectorModel.ScheduledForDeletion)

	validatedConfigJson = json.RawMessage(b)
	m.connectors[connectorModel.ID.String()] = connector{
		plugin:              plugin,
		config:              config,
		validatedConfigJson: validatedConfigJson,
	}
	return config.Name, validatedConfigJson, nil
}

func (m *manager) Unload(connectorID models.ConnectorID) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	_, ok := m.connectors[connectorID.String()]
	if !ok {
		// Nothing to do
		return
	}

	delete(m.connectors, connectorID.String())
}

func (m *manager) Get(connectorID models.ConnectorID) (models.Plugin, error) {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()

	c, ok := m.connectors[connectorID.String()]
	if !ok {
		return nil, fmt.Errorf("%s: %w", connectorID.String(), ErrNotFound)
	}

	return c.plugin, nil
}

func (m *manager) GetConfig(connectorID models.ConnectorID) (models.Config, error) {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()

	c, ok := m.connectors[connectorID.String()]
	if !ok {
		return models.Config{}, fmt.Errorf("%s: %w", connectorID.String(), ErrNotFound)
	}

	return c.config, nil
}

var _ Manager = &manager{}
