package connectors_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigurer(t *testing.T) {
	t.Parallel()

	defaultPeriod := 13 * time.Minute
	minimumPeriod := 5 * time.Minute

	configurer, err := connectors.NewConfigurer(defaultPeriod, minimumPeriod)
	require.NoError(t, err)
	assert.Equal(t, defaultPeriod, configurer.DefaultConfig().PollingPeriod)

	_, err = connectors.NewConfigurer(minimumPeriod-time.Second, minimumPeriod)
	require.Error(t, err)
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	defaultPeriod := 13 * time.Minute
	minimumPeriod := 5 * time.Minute

	configurer, err := connectors.NewConfigurer(defaultPeriod, minimumPeriod)
	require.NoError(t, err)

	validConfig := models.Config{Name: "name", PollingPeriod: 7 * time.Minute}
	err = configurer.Validate(validConfig)
	assert.NoError(t, err)

	invalidConfig := models.Config{Name: "name2", PollingPeriod: 3 * time.Minute}
	err = configurer.Validate(invalidConfig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, connectors.ErrPollingPeriod)

	invalidConfig = models.Config{Name: "n", PollingPeriod: 10 * time.Minute}
	err = configurer.Validate(invalidConfig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidConfig)
}
