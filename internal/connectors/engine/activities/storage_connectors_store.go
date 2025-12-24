package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConnectorsStore(ctx context.Context, connector models.Connector, oldConnectorID *models.ConnectorID) error {
	decryptedConfig, err := a.storage.DecryptRaw(connector.Config)
	if err != nil {
		return temporalStorageError(err)
	}
	connector.Config = decryptedConfig
	return temporalStorageError(a.storage.ConnectorsInstall(ctx, connector, oldConnectorID))
}

var StorageConnectorsStoreActivity = Activities{}.StorageConnectorsStore

func StorageConnectorsStore(ctx workflow.Context, connector models.Connector, oldConnectorID *models.ConnectorID) error {
	return executeActivity(ctx, StorageConnectorsStoreActivity, nil, connector, oldConnectorID)
}
