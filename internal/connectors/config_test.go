package connectors

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineConfigs_BasePollingFlowsThrough(t *testing.T) {
	t.Parallel()

	base := models.Config{
		Name:          "conn-name",
		PollingPeriod: 10 * time.Minute,
	}

	// Plugin configs no longer carry pollingPeriod; only plugin-specific fields.
	pluginCfg := map[string]any{
		"apiKey": "sk_test",
	}

	b, err := combineConfigs(base, pluginCfg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))

	// pollingPeriod always comes from models.Config (user input or server default), never from the plugin.
	assert.Equal(t, "10m0s", out["pollingPeriod"])

	// Base-only fields should be present.
	assert.Equal(t, "conn-name", out["name"])

	// Plugin-specific field preserved.
	assert.Equal(t, "sk_test", out["apiKey"])
}
