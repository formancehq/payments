package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
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

		// Amount, asset and metadata are not carried by the payment status
		// change itself, so we read them back from the payment initiation to
		// keep the emitted adjustment event consistent with the ones produced
		// by the create/reverse workflows.
		pi, err := a.storage.PaymentInitiationsGet(ctx, piID)
		if err != nil {
			return temporalStorageError(err)
		}

		adjustment.Amount = pi.Amount
		adjustment.Asset = &pi.Asset
		adjustment.Metadata = pi.Metadata

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
