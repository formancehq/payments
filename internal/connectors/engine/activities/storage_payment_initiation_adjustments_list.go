package activities

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationAdjustmentsList(ctx context.Context, piID models.PaymentInitiationID, query storage.ListPaymentInitiationAdjustmentsQuery) (*paginate.Cursor[models.PaymentInitiationAdjustment], error) {
	cursor, err := a.storage.PaymentInitiationAdjustmentsList(ctx, piID, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StoragePaymentInitiationAdjustmentsListActivity = Activities{}.StoragePaymentInitiationAdjustmentsList

func StoragePaymentInitiationAdjustmentsList(ctx workflow.Context, piID models.PaymentInitiationID, query storage.ListPaymentInitiationAdjustmentsQuery) (*paginate.Cursor[models.PaymentInitiationAdjustment], error) {
	ret := paginate.Cursor[models.PaymentInitiationAdjustment]{}
	if err := executeActivity(ctx, StoragePaymentInitiationAdjustmentsListActivity, &ret, piID, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
