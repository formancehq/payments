package activities

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsList(ctx context.Context, query storage.ListAccountsQuery) (*paginate.Cursor[models.Account], error) {
	cursor, err := a.storage.AccountsList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StorageAccountsListActivity = Activities{}.StorageAccountsList

func StorageAccountsList(ctx workflow.Context, query storage.ListAccountsQuery) (*paginate.Cursor[models.Account], error) {
	ret := paginate.Cursor[models.Account]{}
	if err := executeActivity(ctx, StorageAccountsListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
