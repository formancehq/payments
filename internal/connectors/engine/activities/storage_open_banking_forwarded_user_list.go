package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersList(ctx context.Context, query storage.ListOpenBankingForwardedUserQuery) (*bunpaginate.Cursor[models.OpenBankingForwardedUser], error) {
	cursor, err := a.storage.OpenBankingForwardedUserList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StorageOpenBankingForwardedUsersListActivity = Activities{}.StorageOpenBankingForwardedUsersList

func StorageOpenBankingForwardedUsersList(ctx workflow.Context, query storage.ListOpenBankingForwardedUserQuery) (*bunpaginate.Cursor[models.OpenBankingForwardedUser], error) {
	ret := bunpaginate.Cursor[models.OpenBankingForwardedUser]{}
	if err := executeActivity(ctx, StorageOpenBankingForwardedUsersListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
