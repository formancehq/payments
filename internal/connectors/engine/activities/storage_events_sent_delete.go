package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageEventsSentDelete(ctx context.Context, connectorID models.ConnectorID) error {
	return a.batchDeleteWithHeartbeat(ctx, connectorID, a.storage.EventsSentDeleteFromConnectorIDBatch, "deleting events_sent")
}

var StorageEventsSentDeleteActivity = Activities{}.StorageEventsSentDelete

func StorageEventsSentDelete(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageEventsSentDeleteActivity, nil, connectorID)
}
