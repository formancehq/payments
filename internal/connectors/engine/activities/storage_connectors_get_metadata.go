package activities

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// StorageConnectorsGetMetadata is similar to StorageConnectorsGet but returns
// only the connectorID, the connector's PollingPeriod and ScheduledForDeletion.
// It still relies on a.storage.ConnectorsGet under the hood.
func (a Activities) StorageConnectorsGetMetadata(ctx context.Context, connectorID models.ConnectorID) (*models.ConnectorMetadata, error) {
	connector, err := a.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}

	// Default polling period from models.DefaultConfig
	polling := models.Config{}.PollingPeriod

	// Try to extract pollingPeriod from the stored connector config payload
	if len(connector.Config) > 0 {
		var cfg models.Config
		if err := json.Unmarshal(connector.Config, &cfg); err == nil && cfg.PollingPeriod > 0 {
			polling = cfg.PollingPeriod
		}
	}

	return &models.ConnectorMetadata{
		ConnectorID:          connector.ID,
		PollingPeriod:        polling,
		ScheduledForDeletion: connector.ScheduledForDeletion,
		Provider:             connector.Provider,
	}, nil
}

var StorageConnectorsGetMetadataActivity = Activities{}.StorageConnectorsGetMetadata

func StorageConnectorsGetMetadata(ctx workflow.Context, connectorID models.ConnectorID) (*models.ConnectorMetadata, error) {
	var out models.ConnectorMetadata
	if err := executeActivity(ctx, StorageConnectorsGetMetadataActivity, &out, connectorID); err != nil {
		return nil, err
	}
	return &out, nil
}
