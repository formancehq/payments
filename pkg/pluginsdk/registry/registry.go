package registry

import (
    "encoding/json"

    "github.com/formancehq/go-libs/v3/logging"
    internalregistry "github.com/formancehq/payments/internal/connectors/plugins/registry"
    internalmodels "github.com/formancehq/payments/internal/models"
)

// RegisterPlugin exposes the internal registry for external connectors via the SDK path.
// It keeps the same signature so existing plugins can register seamlessly.
func RegisterPlugin(
    provider string,
    pluginType internalmodels.PluginType,
    createFunc func(internalmodels.ConnectorID, string, logging.Logger, json.RawMessage) (internalmodels.Plugin, error),
    capabilities []internalmodels.Capability,
    conf any,
) {
    internalregistry.RegisterPlugin(provider, pluginType, createFunc, capabilities, conf)
}

// GetPluginType forwards to the internal registry.
func GetPluginType(provider string) (internalmodels.PluginType, error) {
    return internalregistry.GetPluginType(provider)
}

// GetCapabilities forwards to the internal registry.
func GetCapabilities(provider string) ([]internalmodels.Capability, error) {
    return internalregistry.GetCapabilities(provider)
}

// GetConfigs forwards to the internal registry.
func GetConfigs(debug bool) internalregistry.Configs {
    return internalregistry.GetConfigs(debug)
}

// GetConfig forwards to the internal registry.
func GetConfig(provider string) (internalregistry.Config, error) {
    return internalregistry.GetConfig(provider)
}

// DummyPSPName re-exports the internal constant for convenience.
const DummyPSPName = internalregistry.DummyPSPName

