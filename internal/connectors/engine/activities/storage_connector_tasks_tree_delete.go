package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConnectorTasksTreeDelete(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.ConnectorTasksTreeDeleteFromConnectorID(ctx, connectorID))
}

var StorageConnectorTasksTreeDeleteActivity = Activities{}.StorageConnectorTasksTreeDelete

func StorageConnectortTasksTreeDelete(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageConnectorTasksTreeDeleteActivity, nil, connectorID)
}
