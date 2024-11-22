package plugins

import (
	"encoding/json"
	"fmt"
	"sync"

	registeredPlugins "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("plugin not found")
)

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package plugins . Plugins
type Plugins interface {
	RegisterPlugin(connectorID models.ConnectorID, config models.Config, rawConfig json.RawMessage) error
	UnregisterPlugin(connectorID models.ConnectorID) error
	GetConfig(connectorID models.ConnectorID) (models.Config, error)
	Get(connectorID models.ConnectorID) (models.Plugin, error)
}

// Will start, hold, manage and stop *Plugins
type plugins struct {
	plugins map[string]pluginInformation
	rwMutex sync.RWMutex

	// used to pass flags to plugins
	rawFlags      []string
	debug         bool
	jsonFormatter bool
}

type pluginInformation struct {
	client       models.Plugin
	capabilities map[models.Capability]struct{}
	config       models.Config
}

func New(
	rawFlags []string,
	debug bool,
	jsonFormatter bool,
) *plugins {
	return &plugins{
		plugins:       make(map[string]pluginInformation),
		rawFlags:      rawFlags,
		debug:         debug,
		jsonFormatter: jsonFormatter,
	}
}

func (p *plugins) RegisterPlugin(connectorID models.ConnectorID, config models.Config, rawConfig json.RawMessage) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	// Check if plugin is already installed
	_, ok := p.plugins[connectorID.String()]
	if ok {
		return nil
	}

	plugin, err := registeredPlugins.GetPlugin(connectorID.Provider, rawConfig)
	if err != nil {
		return fmt.Errorf("%w: %w", err, ErrNotFound)
	}

	p.plugins[connectorID.String()] = pluginInformation{
		client:       plugin,
		capabilities: make(map[models.Capability]struct{}),
		config:       config,
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
