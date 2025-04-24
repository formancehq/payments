package models_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("missing name", func(t *testing.T) {
		config := models.Config{}
		err := config.Validate()
		require.Error(t, err)
		require.Equal(t, errors.New("name is required"), err)
	})

	t.Run("invalid polling period", func(t *testing.T) {
		config := models.Config{
			Name:          "test",
			PollingPeriod: 2 * time.Second,
		}
		err := config.Validate()
		require.Error(t, err)
		require.Equal(t, errors.New("polling period must be at least 30 seconds"), err)
	})

	t.Run("valid config", func(t *testing.T) {
		config := models.Config{
			Name:          "test",
			PollingPeriod: 30 * time.Second,
		}
		err := config.Validate()
		// Then
			require.NoError(t, err)
	})
}

func TestConfigMarshalJSON(t *testing.T) {
	t.Parallel()

	config := models.Config{
		Name:          "test-config",
		PollingPeriod: 5 * time.Minute,
		PageSize:      50,
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
	assert.Equal(t, float64(50), jsonMap["pageSize"])
}

func TestConfigUnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "5m",
			"pageSize": 50
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)
		
		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 5*time.Minute, config.PollingPeriod)
		assert.Equal(t, 50, config.PageSize)
	})

	t.Run("default polling period", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pageSize": 50
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)
		
		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 2*time.Minute, config.PollingPeriod) // Default value
		assert.Equal(t, 50, config.PageSize)
	})

	t.Run("invalid polling period", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "invalid",
			"pageSize": 50
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)
		// When/Then
		require.Error(t, err)
	})

	t.Run("zero values", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config",
			"pollingPeriod": "0s",
			"pageSize": 0
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)
		
		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 0*time.Second, config.PollingPeriod) // Zero is allowed
		assert.Equal(t, 0, config.PageSize) // Zero is allowed
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{invalid json}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)
		// When/Then
		require.Error(t, err)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := models.DefaultConfig()

	assert.Equal(t, 2*time.Minute, config.PollingPeriod)
	assert.Equal(t, 25, config.PageSize)
	assert.Empty(t, config.Name)
}
