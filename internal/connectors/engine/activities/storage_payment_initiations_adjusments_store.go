package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationsAdjustmentsStore(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
	return a.storage.PaymentInitiationAdjustmentsUpsert(ctx, adj)
}

var StoragePaymentInitiationsAdjustmentsStoreActivity = Activities{}.StoragePaymentInitiationsAdjustmentsStore

func StoragePaymentInitiationsAdjustmentsStore(ctx workflow.Context, adj models.PaymentInitiationAdjustment) error {
	return executeActivity(ctx, StoragePaymentInitiationsAdjustmentsStoreActivity, nil, adj)
}
