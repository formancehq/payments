package activities

import (
	context "context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDelete(ctx context.Context, id models.PaymentID) error {
	return temporalStorageError(a.storage.PaymentsDelete(ctx, id))
}

var StoragePaymentsDeleteActivity = Activities{}.StoragePaymentsDelete

func StoragePaymentsDelete(ctx workflow.Context, id models.PaymentID) error {
	return executeActivity(ctx, StoragePaymentsDeleteActivity, nil, id)
}
