package registry

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/connector"
	pkgregistry "github.com/formancehq/payments/pkg/registry"
)

// DummyPSPName is an alias to pkg/registry for backward compatibility.
const DummyPSPName = pkgregistry.DummyPSPName

// PluginCreateFunction is an alias to pkg/registry for backward compatibility.
type PluginCreateFunction = pkgregistry.PluginCreateFunction

// ErrPluginNotFound is an alias to pkg/registry for backward compatibility.
var ErrPluginNotFound = pkgregistry.ErrPluginNotFound

// RegisterPlugin delegates to pkg/registry.RegisterPlugin.
// This allows connectors to use either import path.
func RegisterPlugin(
	provider string,
	pluginType models.PluginType,
	createFunc PluginCreateFunction,
	capabilities []models.Capability,
	conf any,
	pageSize uint64,
) {
	// Convert models types to connector types (they're the same underlying types)
	connectorCaps := make([]connector.Capability, len(capabilities))
	for i, c := range capabilities {
		connectorCaps[i] = connector.Capability(c)
	}

	pkgregistry.RegisterPlugin(provider, connector.PluginType(pluginType), createFunc, connectorCaps, conf, pageSize)
}

// GetPlugin retrieves a plugin from the registry, creates it, and wraps it with OTel tracing.
func GetPlugin(connectorID models.ConnectorID, logger logging.Logger, provider string, connectorName string, rawConfig json.RawMessage) (models.Plugin, error) {
	createFunc, _, err := pkgregistry.GetPluginFactory(provider)
	if err != nil {
		return nil, err
	}

	p, err := createFunc(connector.ConnectorID(connectorID), connectorName, logger, rawConfig)
	if err != nil {
		return nil, translateError(err)
	}

	// Wrap with OTel tracing (internal only)
	return New(connectorID, logger, p), nil
}

// GetPluginType delegates to pkg/registry.
func GetPluginType(provider string) (models.PluginType, error) {
	pt, err := pkgregistry.GetPluginType(provider)
	return models.PluginType(pt), err
}

// GetCapabilities delegates to pkg/registry.
func GetCapabilities(provider string) ([]models.Capability, error) {
	caps, err := pkgregistry.GetCapabilities(provider)
	if err != nil {
		return nil, err
	}
	// Convert connector.Capability to models.Capability
	result := make([]models.Capability, len(caps))
	for i, c := range caps {
		result[i] = models.Capability(c)
	}
	return result, nil
}

// GetConfigs delegates to pkg/registry.
func GetConfigs(debug bool) Configs {
	pkgConfigs := pkgregistry.GetConfigs(debug)
	result := make(Configs)
	for k, v := range pkgConfigs {
		result[k] = Config(v)
	}
	return result
}

// GetConfig delegates to pkg/registry.
func GetConfig(provider string) (Config, error) {
	cfg, err := pkgregistry.GetConfig(provider)
	return Config(cfg), err
}

// GetPageSize delegates to pkg/registry.
func GetPageSize(provider string) (uint64, error) {
	return pkgregistry.GetPageSize(provider)
}
