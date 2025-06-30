package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDelete(ctx context.Context, id models.AccountID) error {
	return temporalStorageError(a.storage.AccountsDelete(ctx, id))
}

var StorageAccountsDeleteActivity = Activities{}.StorageAccountsDelete

func StorageAccountsDelete(ctx workflow.Context, id models.AccountID) error {
	return executeActivity(ctx, StorageAccountsDeleteActivity, nil, id)
}
