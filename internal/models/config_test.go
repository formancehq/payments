package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigMarshalJSON(t *testing.T) {
	t.Parallel()

	config := models.Config{
		Name:          "test-config",
		PollingPeriod: 5 * time.Minute,
	}

	data, err := json.Marshal(config)
	// Then
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	// Then
	require.NoError(t, err)

	assert.Equal(t, "test-config", jsonMap["name"])
	assert.Equal(t, "5m0s", jsonMap["pollingPeriod"])
}

func TestConfigUnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "5m"
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)

		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 5*time.Minute, config.PollingPeriod)
	})

	t.Run("invalid polling period", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "invalid"
		}`

		var config models.Config

		err := json.Unmarshal([]byte(jsonData), &config)

		// Then
		require.Error(t, err)
	})

	t.Run("zero values", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "0s"
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)

		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 0*time.Second, config.PollingPeriod) // Zero is allowed
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{invalid json}`

		var config models.Config

		err := json.Unmarshal([]byte(jsonData), &config)

		// Then
		require.Error(t, err)
	})
}
