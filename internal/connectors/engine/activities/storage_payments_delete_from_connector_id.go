package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromConnectorID(ctx, connectorID))
}

var StoragePaymentsDeleteFromConnectorIDActivity = Activities{}.StoragePaymentsDeleteFromConnectorID

func StoragePaymentsDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromConnectorIDActivity, nil, connectorID)
}
