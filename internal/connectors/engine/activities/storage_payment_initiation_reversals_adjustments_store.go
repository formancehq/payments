package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationReversalsAdjustmentsStore(ctx context.Context, adj models.PaymentInitiationReversalAdjustment) error {
	return temporalStorageError(a.storage.PaymentInitiationReversalAdjustmentsUpsert(ctx, adj))
}

var StoragePaymentInitiationReversalsAdjustmentsStoreActivity = Activities{}.StoragePaymentInitiationReversalsAdjustmentsStore

func StoragePaymentInitiationReversalsAdjustmentsStore(ctx workflow.Context, adj models.PaymentInitiationReversalAdjustment) error {
	return executeActivity(ctx, StoragePaymentInitiationReversalsAdjustmentsStoreActivity, nil, adj)
}
