package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationUpdateFromPayment(ctx context.Context, status models.PaymentStatus, createdAt time.Time, paymentID models.PaymentID) error {
	piIDs, err := a.storage.PaymentInitiationIDsListFromPaymentID(ctx, paymentID)
	if err != nil {
		return temporalStorageError(err)
	}

	for _, piID := range piIDs {
		adjustment := models.FromPaymentDataToPaymentInitiationAdjustment(
			status,
			createdAt,
			piID,
		)

		if adjustment == nil {
			continue
		}

		if err := a.storage.PaymentInitiationAdjustmentsUpsert(ctx, *adjustment); err != nil {
			return err
		}
	}
	return nil
}

var StoragePaymentInitiationUpdateFromPaymentActivity = Activities{}.StoragePaymentInitiationUpdateFromPayment

func StoragePaymentInitiationUpdateFromPayment(ctx workflow.Context, status models.PaymentStatus, createdAt time.Time, paymentID models.PaymentID) error {
	if err := executeActivity(ctx, StoragePaymentInitiationUpdateFromPayment, nil, status, createdAt, paymentID); err != nil {
		return err
	}
	return nil
}
