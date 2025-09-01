package connectors

import (
	"encoding/json"
	"io"
	"testing"

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

	logger := logging.NewDefaultLogger(io.Discard, false, false, false)
	manager := NewManager(logger, false)

	tests := map[string]struct {
		provider    string
		config      models.Config
		rawConfig   json.RawMessage
		expectError bool
	}{
		"unregistered plugin provider": {
			provider:    "test",
			config:      models.Config{},
			rawConfig:   json.RawMessage(`{}`),
			expectError: true,
		},
		"invalid config for provider": {
			provider:    registry.DummyPSPName,
			config:      models.Config{},
			rawConfig:   json.RawMessage(`{}`),
			expectError: true,
		},
		"successful load": {
			provider:    registry.DummyPSPName,
			config:      models.DefaultConfig(),
			rawConfig:   json.RawMessage(`{"directory":"/tmp"}`),
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: tt.provider}
			_, err := manager.Load(connectorID, tt.provider, name, tt.config, tt.rawConfig, false)
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
				assert.Equal(t, tt.config, config)

				plugin, err := manager.Get(connectorID)
				assert.NoError(t, err)
				assert.Equal(t, name, plugin.Name())
			}
		})
	}
}

func TestManager_Unload(t *testing.T) {
	t.Parallel()

	logger := logging.NewDefaultLogger(io.Discard, false, false, false)
	manager := NewManager(logger, false)

	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: registry.DummyPSPName}
	manager.connectors[connectorID.String()] = connector{}

	manager.Unload(connectorID)
	assert.NotContains(t, manager.connectors, connectorID.String())
}
