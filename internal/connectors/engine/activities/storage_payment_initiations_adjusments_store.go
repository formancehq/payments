package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) PaymentInitiationsAdjustmentsStore(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
	return a.storage.PaymentInitiationAdjustmentsUpsert(ctx, adj)
}

var PaymentInitiationsAdjustmentsStoreActivity = Activities{}.PaymentInitiationsAdjustmentsStore

func PaymentInitiationsAdjustmentsStore(ctx workflow.Context, adj models.PaymentInitiationAdjustment) error {
	return executeActivity(ctx, PaymentInitiationsAdjustmentsStoreActivity, nil, adj)
}
