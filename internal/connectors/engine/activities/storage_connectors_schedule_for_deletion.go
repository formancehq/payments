package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConnectorsScheduleForDeletion(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.ConnectorsScheduleForDeletion(ctx, connectorID))
}

var StorageConnectorsScheduleForDeletionActivity = Activities{}.StorageConnectorsScheduleForDeletion

func StorageConnectorsScheduleForDeletion(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageConnectorsScheduleForDeletionActivity, nil, connectorID)
}
