package activities

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConnectorsStore(ctx context.Context, connector models.Connector, oldConnectorID *models.ConnectorID) error {
	decryptedConfig, err := a.storage.DecryptRaw(ctx, connector.Config)
	if err != nil {
		if errors.Is(err, storage.ErrNotEncrypted) {
			// payload is already plain JSON; leave as-is
		} else {
			return temporalStorageError(err)
		}
	} else {
		connector.Config = decryptedConfig
	}
	return temporalStorageError(a.storage.ConnectorsInstall(ctx, connector, oldConnectorID))
}

var StorageConnectorsStoreActivity = Activities{}.StorageConnectorsStore

func StorageConnectorsStore(ctx workflow.Context, connector models.Connector, oldConnectorID *models.ConnectorID) error {
	return executeActivity(ctx, StorageConnectorsStoreActivity, nil, connector, oldConnectorID)
}
