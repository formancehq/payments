package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageSchedulesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.SchedulesDeleteFromConnectorID(ctx, connectorID))
}

var StorageSchedulesDeleteFromConnectorIDActivity = Activities{}.StorageSchedulesDeleteFromConnectorID

func StorageSchedulesDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageSchedulesDeleteFromConnectorIDActivity, nil, connectorID)
}
