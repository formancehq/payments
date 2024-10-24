package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error) {
	account, err := a.storage.AccountsGet(ctx, id)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return account, nil
}

var StorageAccountsGetActivity = Activities{}.StorageAccountsGet

func StorageAccountsGet(ctx workflow.Context, id models.AccountID) (*models.Account, error) {
	ret := models.Account{}
	if err := executeActivity(ctx, StorageAccountsGetActivity, &ret, id); err != nil {
		return nil, err
	}
	return &ret, nil
}
