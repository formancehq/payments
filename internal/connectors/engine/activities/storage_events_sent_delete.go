package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageEventsSentDelete(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.EventsSentDeleteFromConnectorID(ctx, connectorID))
}

var StorageEventsSentDeleteActivity = Activities{}.StorageEventsSentDelete

func StorageEventsSentDelete(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageEventsSentDeleteActivity, nil, connectorID)
}
