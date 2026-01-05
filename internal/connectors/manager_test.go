package connectors

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// force load of plugins to registry
	_ "github.com/formancehq/payments/internal/connectors/plugins/public"
)

func TestManager_Load(t *testing.T) {
	t.Parallel()

	minimumPollingPeriod := time.Second
	defaultPollingPeriod := 3 * time.Minute
	logger := logging.NewDefaultLogger(io.Discard, false, false, false)

	tests := map[string]struct {
		provider               string
		expectedConfig         models.Config
		rawConfig              json.RawMessage
		expectError            bool
		strictMode             bool
		isScheduledForDeletion bool
	}{
		"unregistered plugin provider": {
			provider:       "test",
			expectedConfig: models.Config{},
			rawConfig:      json.RawMessage(`{}`),
			expectError:    true,
			strictMode:     true,
		},
		"invalid config for provider": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{},
			rawConfig:      json.RawMessage(`{}`),
			expectError:    true,
			strictMode:     true,
		},
		"provider config present - base config error": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{},
			rawConfig:      json.RawMessage(`{"directory":"/tmp"}`),
			expectError:    true,
			strictMode:     true,
		},
		"base config present - provider specific config error": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{},
			rawConfig:      json.RawMessage(`{"name":"base config present - provider specific config error"}`),
			expectError:    true,
			strictMode:     true,
		},
		"polling period issues ignored when not in strict mode": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{Name: "polling period issues ignored when not in strict mode", PollingPeriod: time.Second},
			rawConfig:      json.RawMessage(`{"name":"polling period issues ignored when not in strict mode","pollingPeriod":"1s","directory":"/tmp"}`),
			expectError:    false,
			strictMode:     false,
		},
		"successful load": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{Name: "successful load", PollingPeriod: 40 * time.Minute},
			rawConfig:      json.RawMessage(`{"name":"successful load","directory":"/tmp","pollingPeriod":"40m"}`),
			expectError:    false,
			strictMode:     true,
		},
		"polling period is set to default when missing": {
			provider:       registry.DummyPSPName,
			expectedConfig: models.Config{Name: "polling period is set to default when missing", PollingPeriod: defaultPollingPeriod},
			rawConfig:      json.RawMessage(`{"name":"polling period is set to default when missing","directory":"/tmp"}`),
			expectError:    false,
			strictMode:     true,
		},
		"connector is scheduled for deletion": {
			provider:               registry.DummyPSPName,
			expectedConfig:         models.Config{Name: "connector is scheduled for deletion", PollingPeriod: 40 * time.Minute},
			rawConfig:              json.RawMessage(`{"name":"connector is scheduled for deletion","directory":"/tmp","pollingPeriod":"40m"}`),
			expectError:            false,
			strictMode:             false,
			isScheduledForDeletion: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewManager(logger, false, defaultPollingPeriod, minimumPollingPeriod)
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: tt.provider}
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Provider: tt.provider,
				},
				Config:               tt.rawConfig,
				ScheduledForDeletion: tt.isScheduledForDeletion,
			}
			returnedName, _, err := manager.Load(connector, false, tt.strictMode)
			if tt.expectError {
				require.Error(t, err)

				_, err := manager.GetConfig(connectorID)
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrNotFound)

				_, err = manager.Get(connectorID)
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrNotFound)
			} else {
				require.NoError(t, err)
				assert.Contains(t, manager.connectors, connectorID.String())

				config, err := manager.GetConfig(connectorID)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedConfig, config)

				plugin, err := manager.Get(connectorID)
				assert.NoError(t, err)
				assert.Equal(t, name, plugin.Name())
				assert.Equal(t, returnedName, plugin.Name())
				assert.Equal(t, tt.isScheduledForDeletion, plugin.IsScheduledForDeletion())
			}
		})
	}
}

func TestManager_Unload(t *testing.T) {
	t.Parallel()

	minimumPollingPeriod := time.Second
	logger := logging.NewDefaultLogger(io.Discard, false, false, false)
	manager := NewManager(logger, false, time.Minute, minimumPollingPeriod)

	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: registry.DummyPSPName}
	manager.connectors[connectorID.String()] = connector{}

	manager.Unload(connectorID)
	assert.NotContains(t, manager.connectors, connectorID.String())
}
