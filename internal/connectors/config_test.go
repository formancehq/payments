package connectors

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineConfigs_PluginPrecedence(t *testing.T) {
	t.Parallel()

	base := models.Config{
		Name:          "conn-name",
		PollingPeriod: 10 * time.Minute, // lower than plugin
		PageSize:      50,
	}

	pluginCfg := map[string]any{
		"apiKey":        "sk_test",
		"pollingPeriod": "20m0s", // plugin-normalized value should win
	}

	b, err := combineConfigs(base, pluginCfg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))

	// Plugin value should take precedence
	assert.Equal(t, "20m0s", out["pollingPeriod"])

	// Base-only fields should be present
	assert.Equal(t, "conn-name", out["name"])
	// pageSize from base since plugin didn't set it
	assert.Equal(t, float64(50), out["pageSize"]) // numbers become float64 via json

	// Plugin specific field preserved
	assert.Equal(t, "sk_test", out["apiKey"])
}
