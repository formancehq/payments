package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgesList(ctx context.Context, query storage.ListPSUBankBridgesQuery) (*bunpaginate.Cursor[models.PSUBankBridge], error) {
	cursor, err := a.storage.PSUBankBridgesList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StoragePSUBankBridgesListActivity = Activities{}.StoragePSUBankBridgesList

func StoragePSUBankBridgesList(ctx workflow.Context, query storage.ListPSUBankBridgesQuery) (*bunpaginate.Cursor[models.PSUBankBridge], error) {
	ret := bunpaginate.Cursor[models.PSUBankBridge]{}
	if err := executeActivity(ctx, StoragePSUBankBridgesListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
