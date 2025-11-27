package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("missing name", func(t *testing.T) {
		config := models.Config{
			PollingPeriod: 40 * time.Minute,
		}
		err := config.Validate()
		require.Error(t, err)
		vErrs, ok := err.(validator.ValidationErrors)
		require.True(t, ok)
		require.Len(t, vErrs, 1)
		assert.Equal(t, "Name", vErrs[0].Field())
		assert.Equal(t, "required", vErrs[0].Tag())
	})

	t.Run("polling period out of bounds", func(t *testing.T) {
		tests := map[string]struct {
			val time.Duration
			tag string
		}{
			"too short": {
				val: 2 * time.Second,
				tag: "gte",
			},
			"too long": {
				val: (24 * time.Hour) + time.Second,
				tag: "lte",
			},
		}

		for name, c := range tests {
			t.Run(name, func(t *testing.T) {
				config := models.Config{
					Name:          "test",
					PollingPeriod: c.val,
				}
				err := config.Validate()
				require.Error(t, err)
				vErrs, ok := err.(validator.ValidationErrors)
				require.True(t, ok)
				require.Len(t, vErrs, 1)
				assert.Equal(t, "PollingPeriod", vErrs[0].Field())
				assert.Equal(t, c.tag, vErrs[0].Tag())
			})
		}
	})

	t.Run("valid config", func(t *testing.T) {
		config := models.Config{
			Name:          "test",
			PollingPeriod: 30 * time.Minute,
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

	t.Run("default polling period", func(t *testing.T) {
		t.Parallel()
		// Given

		jsonData := `{
			"name": "test-config"
		}`

		var config models.Config
		err := json.Unmarshal([]byte(jsonData), &config)

		// Then
		require.NoError(t, err)

		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, 30*time.Minute, config.PollingPeriod) // Default value
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

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := models.DefaultConfig()

	assert.Equal(t, 30*time.Minute, config.PollingPeriod)
	assert.Empty(t, config.Name)
}
