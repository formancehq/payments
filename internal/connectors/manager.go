package connectors

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/formancehq/go-libs/v3/logging"
	pluginserrors "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

var (
	ErrNotFound         = errors.New("plugin not found")
	ErrValidation       = errors.New("validation error")
	ErrInvalidOperation = errors.New("invalid operation")
)

//go:generate mockgen -source manager.go -destination manager_generated.go -package connectors . Manager
type Manager interface {
	Load(models.ConnectorID, string, string, models.Config, json.RawMessage, bool) error
	Unload(models.ConnectorID)
	GetConfig(models.ConnectorID) (models.Config, error)
	Get(models.ConnectorID) (models.Plugin, error)
}

// Will start, hold, manage and stop Connectors
type manager struct {
	logger logging.Logger

	connectors map[string]connector
	rwMutex    sync.RWMutex

	debug bool
}

type connector struct {
	plugin models.Plugin
	config models.Config
}

func NewManager(
	logger logging.Logger,
	debug bool,
) *manager {
	return &manager{
		logger:     logger,
		connectors: make(map[string]connector),
		debug:      debug,
	}
}

func (p *manager) Load(
	connectorID models.ConnectorID,
	provider string,
	connectorName string,
	config models.Config,
	rawConfig json.RawMessage,
	updateExisting bool,
) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	// Check if plugin is already installed
	_, ok := p.connectors[connectorID.String()]
	if ok && !updateExisting {
		return nil
	}

	plugin, err := registry.GetPlugin(connectorID, p.logger, provider, connectorName, rawConfig)
	switch {
	case errors.Is(err, pluginserrors.ErrNotImplemented),
		errors.Is(err, pluginserrors.ErrInvalidClientRequest):
		return fmt.Errorf("%w: %w", err, ErrValidation)
	case err != nil:
		return err
	}

	p.connectors[connectorID.String()] = connector{
		plugin: plugin,
		config: config,
	}

	return nil
}

func (p *manager) Unload(connectorID models.ConnectorID) {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	_, ok := p.connectors[connectorID.String()]
	if !ok {
		// Nothing to do
		return
	}

	delete(p.connectors, connectorID.String())
}

func (p *manager) Get(connectorID models.ConnectorID) (models.Plugin, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	c, ok := p.connectors[connectorID.String()]
	if !ok {
		return nil, fmt.Errorf("%s: %w", connectorID.String(), ErrNotFound)
	}

	return c.plugin, nil
}

func (p *manager) GetConfig(connectorID models.ConnectorID) (models.Config, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	c, ok := p.connectors[connectorID.String()]
	if !ok {
		return models.Config{}, fmt.Errorf("%s: %w", connectorID.String(), ErrNotFound)
	}

	return c.config, nil
}

var _ Manager = &manager{}
