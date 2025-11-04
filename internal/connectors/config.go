package connectors

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

func combineConfigs(baseConfig models.Config, pluginConfig models.PluginInternalConfig) ([]byte, error) {
	baseJSON, err := json.Marshal(baseConfig)
	if err != nil {
		return nil, err
	}

	var baseMap map[string]interface{}
	if err := json.Unmarshal(baseJSON, &baseMap); err != nil {
		return nil, err
	}

	pluginJSON, err := json.Marshal(pluginConfig)
	if err != nil {
		return nil, err
	}

	var pluginMap map[string]interface{}
	if err := json.Unmarshal(pluginJSON, &pluginMap); err != nil {
		return nil, err
	}

	if pluginMap == nil {
		pluginMap = make(map[string]interface{}, len(baseMap))
	}

	//Merge maps giving precedence to pluginConfig values
	for key, value := range baseMap {
		_, exists := pluginMap[key]
		if !exists {
			pluginMap[key] = value
			continue
		}
	}

	return json.Marshal(pluginMap)
}
