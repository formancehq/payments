package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageTasksDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.TasksDeleteFromConnectorID(ctx, connectorID))
}

var StorageTasksDeleteFromConnectorIDActivity = Activities{}.StorageTasksDeleteFromConnectorID

func StorageTasksDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageTasksDeleteFromConnectorIDActivity, nil, connectorID)
}
