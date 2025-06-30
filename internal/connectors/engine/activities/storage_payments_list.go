package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsList(ctx context.Context, query storage.ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	cursor, err := a.storage.PaymentsList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StoragePaymentsListActivity = Activities{}.StoragePaymentsList

func StoragePaymentsList(ctx workflow.Context, query storage.ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	ret := bunpaginate.Cursor[models.Payment]{}
	if err := executeActivity(ctx, StoragePaymentsListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
