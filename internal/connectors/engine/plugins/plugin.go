package plugins

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/formancehq/go-libs/v2/logging"
	pluginserrors "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

var (
	ErrNotFound   = errors.New("plugin not found")
	ErrValidation = errors.New("validation error")
)

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package plugins . Plugins
type Plugins interface {
	RegisterPlugin(models.ConnectorID, string, models.Config, json.RawMessage, bool) error
	UnregisterPlugin(models.ConnectorID) error
	GetConfig(models.ConnectorID) (models.Config, error)
	Get(models.ConnectorID) (models.Plugin, error)
}

// Will start, hold, manage and stop *Plugins
type plugins struct {
	logger logging.Logger

	plugins map[string]pluginInformation
	rwMutex sync.RWMutex

	debug bool
}

type pluginInformation struct {
	client models.Plugin
	config models.Config
}

func New(
	logger logging.Logger,
	debug bool,
) *plugins {
	return &plugins{
		logger:  logger,
		plugins: make(map[string]pluginInformation),
		debug:   debug,
	}
}

func (p *plugins) RegisterPlugin(
	connectorID models.ConnectorID,
	connectorName string,
	config models.Config,
	rawConfig json.RawMessage,
	updateExisting bool,
) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	// Check if plugin is already installed
	_, ok := p.plugins[connectorID.String()]
	if ok && !updateExisting {
		return nil
	}

	plugin, err := registry.GetPlugin(p.logger, connectorID.Provider, connectorName, rawConfig)
	switch {
	case errors.Is(err, pluginserrors.ErrNotImplemented),
		errors.Is(err, pluginserrors.ErrInvalidClientRequest):
		return fmt.Errorf("%w: %w", err, ErrValidation)
	case err != nil:
		return err
	}

	p.plugins[connectorID.String()] = pluginInformation{
		client: plugin,
		config: config,
	}

	return nil
}

func (p *plugins) UnregisterPlugin(connectorID models.ConnectorID) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	_, ok := p.plugins[connectorID.String()]
	if !ok {
		// Nothing to do``
		return nil
	}

	delete(p.plugins, connectorID.String())

	return nil
}

func (p *plugins) Get(connectorID models.ConnectorID) (models.Plugin, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	pluginInfo, ok := p.plugins[connectorID.String()]
	if !ok {
		return nil, ErrNotFound
	}

	return pluginInfo.client, nil
}

func (p *plugins) GetConfig(connectorID models.ConnectorID) (models.Config, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	pluginInfo, ok := p.plugins[connectorID.String()]
	if !ok {
		return models.Config{}, ErrNotFound
	}

	return pluginInfo.config, nil
}

var _ Plugins = &plugins{}
