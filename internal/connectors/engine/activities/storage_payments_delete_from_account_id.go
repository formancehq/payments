package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromAccountID(ctx context.Context, accountID models.AccountID) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromAccountID(ctx, accountID))
}

var StoragePaymentsDeleteFromAccountIDActivity = Activities{}.StoragePaymentsDeleteFromAccountID

func StoragePaymentsDeleteFromAccountID(ctx workflow.Context, accountID models.AccountID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromAccountIDActivity, nil, accountID)
}
