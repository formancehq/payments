package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingProviderPSUsList(ctx context.Context, query storage.ListOpenBankingProviderPSUQuery) (*bunpaginate.Cursor[models.OpenBankingProviderPSU], error) {
	cursor, err := a.storage.OpenBankingProviderPSUList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StorageOpenBankingProviderPSUsListActivity = Activities{}.StorageOpenBankingProviderPSUsList

func StorageOpenBankingProviderPSUsList(ctx workflow.Context, query storage.ListOpenBankingProviderPSUQuery) (*bunpaginate.Cursor[models.OpenBankingProviderPSU], error) {
	ret := bunpaginate.Cursor[models.OpenBankingProviderPSU]{}
	if err := executeActivity(ctx, StorageOpenBankingProviderPSUsListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
