package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentInitiationsCreate(ctx context.Context, paymentInitiation models.PaymentInitiation, sendToPSP bool) error {
	waitingForValidationAdjustment := models.PaymentInitiationAdjustment{
		ID: models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: paymentInitiation.ID,
			CreatedAt:           paymentInitiation.CreatedAt,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
		},
		PaymentInitiationID: paymentInitiation.ID,
		CreatedAt:           paymentInitiation.CreatedAt,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}

	if !sendToPSP {
		return newStorageError(s.storage.PaymentInitiationsUpsert(ctx, paymentInitiation, waitingForValidationAdjustment), "cannot create payment initiation")
	}

	if err := s.storage.PaymentInitiationsUpsert(ctx, paymentInitiation, waitingForValidationAdjustment); err != nil {
		return newStorageError(err, "cannot create payment initiation")
	}

	switch paymentInitiation.Type {
	case models.PAYMENT_INITIATION_TYPE_TRANSFER:
		return handleEngineErrors(s.engine.CreateTransfer(ctx, paymentInitiation.ID, 1))
	case models.PAYMENT_INITIATION_TYPE_PAYOUT:
		return handleEngineErrors(s.engine.CreatePayout(ctx, paymentInitiation.ID, 1))
	}

	return nil
}
