package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return a.batchDeleteWithHeartbeat(ctx, connectorID, a.storage.PaymentsDeleteFromConnectorIDBatch, "deleting payments")
}

var StoragePaymentsDeleteFromConnectorIDActivity = Activities{}.StoragePaymentsDeleteFromConnectorID

func StoragePaymentsDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromConnectorIDActivity, nil, connectorID)
}
