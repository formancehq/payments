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
			PollingPeriod: 40 * time.Second,
			PageSize:      30,
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
					PageSize:      30,
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

	t.Run("page size out of bounds", func(t *testing.T) {
		tests := map[string]struct {
			val int
			tag string
		}{
			"too long": {
				val: 151,
				tag: "lte",
			},
		}

		for name, c := range tests {
			t.Run(name, func(t *testing.T) {
				config := models.Config{
					Name:          "test",
					PollingPeriod: time.Minute,
					PageSize:      c.val,
				}
				err := config.Validate()
				require.Error(t, err)
				vErrs, ok := err.(validator.ValidationErrors)
				require.True(t, ok)
				require.Len(t, vErrs, 1)
				assert.Equal(t, "PageSize", vErrs[0].Field())
				assert.Equal(t, c.tag, vErrs[0].Tag())
			})
		}
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

		// Then
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
		assert.Equal(t, 0, config.PageSize)                  // Zero is allowed
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

	assert.Equal(t, 2*time.Minute, config.PollingPeriod)
	assert.Equal(t, 25, config.PageSize)
	assert.Empty(t, config.Name)
}
