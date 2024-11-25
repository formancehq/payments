package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationReversalsDelete(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PaymentInitiationReversalsDeleteFromConnectorID(ctx, connectorID))
}

var StoragePaymentInitiationReversalsDeleteActivity = Activities{}.StoragePaymentInitiationReversalsDelete

func StoragePaymentInitiationReversalsDelete(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentInitiationReversalsDeleteActivity, nil, connectorID)
}
