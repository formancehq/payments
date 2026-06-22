package connectors_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigurer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		def         time.Duration
		min         time.Duration
		expectError bool
	}{
		{"valid", 13 * time.Minute, 5 * time.Minute, false},
		{"default below minimum", 4 * time.Minute, 5 * time.Minute, true},
		{"zero default", 0, 5 * time.Minute, true},
		{"zero minimum", 13 * time.Minute, 0, true},
		{"both zero", 0, 0, true},
		{"negative default", -1 * time.Minute, 5 * time.Minute, true},
		{"negative minimum", 13 * time.Minute, -1 * time.Minute, true},
		{"both negative", -1 * time.Minute, -2 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			configurer, err := connectors.NewConfigurer(tt.def, tt.min)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.def, configurer.DefaultConfig().PollingPeriod)
			}
		})
	}
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
