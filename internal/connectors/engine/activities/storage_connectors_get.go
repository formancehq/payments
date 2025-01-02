package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConnectorsGet(ctx context.Context, connectorID models.ConnectorID) (*models.Connector, error) {
	connector, err := a.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return connector, nil
}

var StorageConnectorsGetActivity = Activities{}.StorageConnectorsGet

func StorageConnectorsGet(ctx workflow.Context, connectorID models.ConnectorID) (*models.Connector, error) {
	var connector models.Connector
	err := executeActivity(ctx, StorageConnectorsGetActivity, &connector, connectorID)
	if err != nil {
		return nil, err
	}
	return &connector, err
}
