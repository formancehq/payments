package plugins

import (
	"encoding/json"
	"errors"

	"github.com/formancehq/payments/internal/models"
)

type PluginCreateFunction func(json.RawMessage) (models.Plugin, error)

var (
	plugins map[string]PluginCreateFunction

	ErrPluginNotFound = errors.New("plugin not found")
)

func RegisterPlugin(name string, createFunc PluginCreateFunction) {
	plugins[name] = createFunc
}

func GetPlugin(name string, rawConfig json.RawMessage) (models.Plugin, error) {
	createFunc, ok := plugins[name]
	if !ok {
		return nil, ErrPluginNotFound
	}

	p, err := createFunc(rawConfig)
	if err != nil {
		return nil, err
	}

	return New(p), nil
}
