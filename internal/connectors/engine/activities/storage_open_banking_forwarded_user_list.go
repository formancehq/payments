package activities

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersList(ctx context.Context, query storage.ListOpenBankingForwardedUserQuery) (*paginate.Cursor[models.OpenBankingForwardedUser], error) {
	cursor, err := a.storage.OpenBankingForwardedUserList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StorageOpenBankingForwardedUsersListActivity = Activities{}.StorageOpenBankingForwardedUsersList

func StorageOpenBankingForwardedUsersList(ctx workflow.Context, query storage.ListOpenBankingForwardedUserQuery) (*paginate.Cursor[models.OpenBankingForwardedUser], error) {
	ret := paginate.Cursor[models.OpenBankingForwardedUser]{}
	if err := executeActivity(ctx, StorageOpenBankingForwardedUsersListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
