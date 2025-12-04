package connectors_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigurer(t *testing.T) {
	t.Parallel()

	defaultPeriod := 13 * time.Minute
	minimumPeriod := 5 * time.Minute

	configurer := connectors.NewConfigurer(defaultPeriod, minimumPeriod)

	assert.Equal(t, defaultPeriod, configurer.DefaultConfig().PollingPeriod)
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	defaultPeriod := 13 * time.Minute
	minimumPeriod := 5 * time.Minute

	configurer := connectors.NewConfigurer(defaultPeriod, minimumPeriod)

	validConfig := models.Config{Name: "name", PollingPeriod: 7 * time.Minute}
	err := configurer.Validate(validConfig)
	assert.NoError(t, err)

	invalidConfig := models.Config{Name: "name2", PollingPeriod: 3 * time.Minute}
	err = configurer.Validate(invalidConfig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidConfig)

	invalidConfig = models.Config{Name: "n", PollingPeriod: 10 * time.Minute}
	err = configurer.Validate(invalidConfig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidConfig)
}
