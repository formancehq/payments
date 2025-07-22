package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromReference(ctx context.Context, reference string, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromReference(ctx, reference, connectorID))
}

var StoragePaymentsDeleteFromReferenceActivity = Activities{}.StoragePaymentsDeleteFromReference

func StoragePaymentsDeleteFromReference(ctx workflow.Context, reference string, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromReferenceActivity, nil, reference, connectorID)
}
