package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type RelatedPayment struct {
	PiID      models.PaymentInitiationID
	PID       models.PaymentID
	CreatedAt time.Time
}

func (a Activities) StoragePaymentInitiationsRelatedPaymentsStore(ctx context.Context, relatedPayment RelatedPayment) error {
	return a.storage.PaymentInitiationRelatedPaymentsUpsert(ctx, relatedPayment.PiID, relatedPayment.PID, relatedPayment.CreatedAt)
}

var StoragePaymentInitiationsRelatedPaymentsStoreActivity = Activities{}.StoragePaymentInitiationsRelatedPaymentsStore

func StoragePaymentInitiationsRelatedPaymentsStore(ctx workflow.Context, piID models.PaymentInitiationID, pID models.PaymentID, createdAt time.Time) error {
	return executeActivity(ctx, StoragePaymentInitiationsRelatedPaymentsStoreActivity, nil, RelatedPayment{
		PiID:      piID,
		PID:       pID,
		CreatedAt: createdAt,
	})
}
