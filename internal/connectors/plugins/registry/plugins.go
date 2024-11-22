package registry

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
)

type PluginCreateFunction func(string, json.RawMessage) (models.Plugin, error)

var (
	pluginsRegistry map[string]PluginCreateFunction = make(map[string]PluginCreateFunction)

	ErrPluginNotFound = errors.New("plugin not found")
)

func RegisterPlugin(name string, createFunc PluginCreateFunction) {
	pluginsRegistry[name] = createFunc
}

func GetPlugin(logger logging.Logger, provider string, connectorName string, rawConfig json.RawMessage) (models.Plugin, error) {
	createFunc, ok := pluginsRegistry[strings.ToLower(provider)]
	if !ok {
		return nil, ErrPluginNotFound
	}

	p, err := createFunc(connectorName, rawConfig)
	if err != nil {
		return nil, err
	}

	return New(logger, p), nil
}
