package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationsDelete(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PaymentInitiationsDeleteFromConnectorID(ctx, connectorID))
}

var StoragePaymentInitiationsDeleteActivity = Activities{}.StoragePaymentInitiationsDelete

func StoragePaymentInitiationsDelete(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentInitiationsDeleteActivity, nil, connectorID)
}
