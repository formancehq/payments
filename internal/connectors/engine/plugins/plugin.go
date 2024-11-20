package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/payments/internal/connectors/grpc"
	"github.com/formancehq/payments/internal/models"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/propagation"
	ggrpc "google.golang.org/grpc"
)

var (
	ErrNotFound = errors.New("plugin not found")
)

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package plugins . Plugins
type Plugins interface {
	RegisterPlugin(connectorID models.ConnectorID, config models.Config) error
	AddCapabilities(connectorID models.ConnectorID, capabilities []models.Capability) error
	GetCapabilities(connectorID models.ConnectorID) (map[models.Capability]struct{}, error)
	GetConfig(connectorID models.ConnectorID) (models.Config, error)
	UnregisterPlugin(connectorID models.ConnectorID) error
	Get(connectorID models.ConnectorID) (models.Plugin, error)
}

// Will start, hold, manage and stop plugins
type plugins struct {
	pluginsPath map[string]string

	plugins map[string]pluginInformation
	rwMutex sync.RWMutex

	// used to pass flags to plugins
	rawFlags      []string
	debug         bool
	jsonFormatter bool
}

type pluginInformation struct {
	client       *plugin.Client
	capabilities map[models.Capability]struct{}
	config       models.Config
}

func New(
	pluginsPath map[string]string,
	rawFlags []string,
	debug bool,
	jsonFormatter bool,
) *plugins {
	return &plugins{
		pluginsPath:   pluginsPath,
		plugins:       make(map[string]pluginInformation),
		rawFlags:      rawFlags,
		debug:         debug,
		jsonFormatter: jsonFormatter,
	}
}

func (p *plugins) RegisterPlugin(connectorID models.ConnectorID, config models.Config) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	// Check if plugin is already installed
	_, ok := p.plugins[connectorID.String()]
	if ok {
		return nil
	}

	pluginPath, ok := p.pluginsPath[connectorID.Provider]
	if !ok {
		return ErrNotFound
	}

	loggerOptions := &hclog.LoggerOptions{
		Name:   fmt.Sprintf("%s-%s", connectorID.Provider, connectorID.String()),
		Output: os.Stdout,
		Level:  hclog.Info,
	}

	if p.debug {
		loggerOptions.Level = hclog.Debug
	}
	if p.jsonFormatter {
		loggerOptions.JSONFormat = true
	}

	dialOpts := []ggrpc.DialOption{
		ggrpc.WithIdleTimeout(0), // disable grpc idle timeout
		ggrpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		ggrpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(
			otelgrpc.WithPropagators(
				propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
			),
		)),
	}

	logger := hclog.New(loggerOptions)
	pc := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  grpc.Handshake(),
		Plugins:          grpc.PluginMap,
		Cmd:              pluginCmd(pluginPath, p.rawFlags, logger),
		GRPCDialOptions:  dialOpts,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           logger,
	})

	p.plugins[connectorID.String()] = pluginInformation{
		client:       pc,
		capabilities: make(map[models.Capability]struct{}),
		config:       config,
	}

	return nil
}

func (p *plugins) AddCapabilities(connectorID models.ConnectorID, capabilities []models.Capability) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	pluginInfo, ok := p.plugins[connectorID.String()]
	if !ok {
		return ErrNotFound
	}

	for _, capability := range capabilities {
		pluginInfo.capabilities[capability] = struct{}{}
	}

	return nil
}

func (p *plugins) UnregisterPlugin(connectorID models.ConnectorID) error {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	pluginInfo, ok := p.plugins[connectorID.String()]
	if !ok {
		// Nothing to do``
		return nil
	}

	// Close the connection
	pluginInfo.client.Kill()

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

	return getPlugin(pluginInfo.client)
}

func (p *plugins) GetCapabilities(connectorID models.ConnectorID) (map[models.Capability]struct{}, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	pluginInfo, ok := p.plugins[connectorID.String()]
	if !ok {
		return nil, ErrNotFound
	}

	return pluginInfo.capabilities, nil
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

func getPlugin(client *plugin.Client) (models.Plugin, error) {
	// Connect via RPC
	conn, err := client.Client()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to plugin")
	}

	raw, err := conn.Dispense("psp")
	if err != nil {
		return nil, errors.Wrap(err, "failed to dispense plugin")
	}

	plugin, ok := raw.(grpc.PSP)
	if !ok {
		return nil, errors.New("failed to cast plugin")
	}

	impl := &impl{
		pluginClient: plugin,
	}

	return impl, nil
}

func pluginCmd(pluginPath string, rawFlags []string, logger hclog.Logger) *exec.Cmd {
	flags := make([]string, 0)
	for _, flag := range rawFlags {
		if strings.Contains(flag, otlpmetrics.OtelMetricsExporterFlag) {
			if strings.HasSuffix(flag, otlpmetrics.StdoutExporter) {
				logger.Debug("reverting to noop exporter as stdout is not supported in plugin", "original_flag", flag)
				continue
			}
		}
		flags = append(flags, flag)
	}

	return exec.Command(pluginPath, flags...)
}

var _ Plugins = &plugins{}
