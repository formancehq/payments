package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationUpdateFromPayment(ctx context.Context, status models.PaymentStatus, createdAt time.Time, paymentID models.PaymentID) error {
	pis, err := a.storage.PaymentInitiationsListFromPaymentID(ctx, paymentID)
	if err != nil {
		return temporalStorageError(err)
	}

	for _, pi := range pis {
		adjustment := models.FromPaymentDataToPaymentInitiationAdjustment(
			status,
			createdAt,
			pi,
		)

		if adjustment == nil {
			continue
		}

		if err := a.storage.PaymentInitiationAdjustmentsUpsert(ctx, *adjustment); err != nil {
			return temporalStorageError(err)
		}
	}
	return nil
}

var StoragePaymentInitiationUpdateFromPaymentActivity = Activities{}.StoragePaymentInitiationUpdateFromPayment

func StoragePaymentInitiationUpdateFromPayment(ctx workflow.Context, status models.PaymentStatus, createdAt time.Time, paymentID models.PaymentID) error {
	if err := executeActivity(ctx, StoragePaymentInitiationUpdateFromPaymentActivity, nil, status, createdAt, paymentID); err != nil {
		return err
	}
	return nil
}
