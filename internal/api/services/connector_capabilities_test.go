package services

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

const testCapabilitiesProvider = "services-capabilities-test"

// The plugin registry is a process-wide singleton. We register a synthetic
// plugin at package init so the services tests can exercise capability lookup
// without pulling in the heavyweight public plugin set. Doing this in init
// (not lazily) keeps the registry map effectively read-only by the time any
// test - including parallel ones - reads it, matching the production
// invariant established by real plugin init() functions.
func init() {
	type testCfg struct {
		Foo string `json:"foo"`
	}
	registry.RegisterPlugin(
		testCapabilitiesProvider,
		models.PluginTypePSP,
		func(_ models.ConnectorID, _ string, _ logging.Logger, _ json.RawMessage) (models.Plugin, error) {
			return nil, nil
		},
		[]models.Capability{models.CAPABILITY_FETCH_ACCOUNTS, models.CAPABILITY_CREATE_TRANSFER},
		testCfg{},
		100,
	)
}

func TestConnectorsCapabilities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := New(storage.NewMockStorage(ctrl), engine.NewMockEngine(ctrl), false)

	caps := s.ConnectorsCapabilities()
	require.NotNil(t, caps)
	assert.Equal(t,
		[]models.Capability{models.CAPABILITY_FETCH_ACCOUNTS, models.CAPABILITY_CREATE_TRANSFER},
		caps[testCapabilitiesProvider],
	)
}

func TestConnectorsCapabilitiesGet(t *testing.T) {
	t.Parallel()

	newService := func(t *testing.T) (*Service, *storage.MockStorage) {
		t.Helper()
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		store := storage.NewMockStorage(ctrl)
		return New(store, engine.NewMockEngine(ctrl), true), store
	}

	t.Run("returns plugin capabilities for known provider", func(t *testing.T) {
		s, store := newService(t)
		connectorID := models.ConnectorID{Provider: testCapabilitiesProvider, Reference: uuid.New()}
		store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(&models.Connector{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: testCapabilitiesProvider},
		}, nil)

		caps, err := s.ConnectorsCapabilitiesGet(t.Context(), connectorID)
		require.NoError(t, err)
		assert.Equal(t,
			[]models.Capability{models.CAPABILITY_FETCH_ACCOUNTS, models.CAPABILITY_CREATE_TRANSFER},
			caps,
		)
	})

	t.Run("wraps storage not-found", func(t *testing.T) {
		s, store := newService(t)
		connectorID := models.ConnectorID{Provider: "missing", Reference: uuid.New()}
		store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(nil, storage.ErrNotFound)

		_, err := s.ConnectorsCapabilitiesGet(t.Context(), connectorID)
		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("translates unknown plugin to ErrNotFound", func(t *testing.T) {
		s, store := newService(t)
		connectorID := models.ConnectorID{Provider: "ghost", Reference: uuid.New()}
		store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(&models.Connector{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "ghost"},
		}, nil)

		_, err := s.ConnectorsCapabilitiesGet(t.Context(), connectorID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
