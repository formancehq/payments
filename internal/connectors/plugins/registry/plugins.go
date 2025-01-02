package registry

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
)

type PluginCreateFunction func(string, json.RawMessage) (models.Plugin, error)

type PluginInformation struct {
	capabilities []models.Capability
	createFunc   PluginCreateFunction
}

var (
	pluginsRegistry map[string]PluginInformation = make(map[string]PluginInformation)

	ErrPluginNotFound = errors.New("plugin not found")
)

func RegisterPlugin(provider string, createFunc PluginCreateFunction, capabilities []models.Capability) {
	pluginsRegistry[provider] = PluginInformation{
		capabilities: capabilities,
		createFunc:   createFunc,
	}
}

func GetPlugin(logger logging.Logger, provider string, connectorName string, rawConfig json.RawMessage) (models.Plugin, error) {
	info, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}

	p, err := info.createFunc(connectorName, rawConfig)
	if err != nil {
		return nil, translateError(err)
	}

	return New(logger, p), nil
}

func GetCapabilities(provider string) ([]models.Capability, error) {
	info, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}

	return info.capabilities, nil
}
