package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestConnectorConfigs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	require.NotNil(t, s.ConnectorsConfigs())
}

func TestConnectorConfig(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	providers := map[string]string{
		"atlar":         "Atlar",
		"bankingcircle": "Bankingcircle",
		"stripe":        "Stripe",
	}

	type descriminator struct {
		Provider string `json:"provider"`
	}

	for provider, expected := range providers {
		connectorID := models.ConnectorID{
			Provider:  provider,
			Reference: uuid.New(),
		}
		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID:       connectorID,
				Provider: provider,
			},
			Config: json.RawMessage(`{}`),
		}
		store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)

		rawConf, err := s.ConnectorsConfig(context.Background(), connectorID)
		require.NoError(t, err)

		var conf descriminator
		err = json.Unmarshal(rawConf, &conf)
		require.NoError(t, err)
		assert.Equal(t, expected, conf.Provider)
	}
}
