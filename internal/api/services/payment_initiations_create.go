package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentInitiationsCreate(ctx context.Context, paymentInitiation models.PaymentInitiation, sendToPSP bool, waitResult bool) (models.Task, error) {
	waitingForValidationAdjustment := models.PaymentInitiationAdjustment{
		ID: models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: paymentInitiation.ID,
			CreatedAt:           paymentInitiation.CreatedAt,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
		},
		PaymentInitiationID: paymentInitiation.ID,
		CreatedAt:           paymentInitiation.CreatedAt,
		Amount:              paymentInitiation.Amount,
		Asset:               &paymentInitiation.Asset,
		Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION,
	}

	if !sendToPSP {
		return models.Task{}, newStorageError(s.storage.PaymentInitiationsUpsert(ctx, paymentInitiation, waitingForValidationAdjustment), "cannot create payment initiation")
	}

	if err := s.storage.PaymentInitiationsUpsert(ctx, paymentInitiation, waitingForValidationAdjustment); err != nil {
		return models.Task{}, newStorageError(err, "cannot create payment initiation")
	}

	switch paymentInitiation.Type {
	case models.PAYMENT_INITIATION_TYPE_TRANSFER:
		task, err := s.engine.CreateTransfer(ctx, paymentInitiation.ID, 1, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	case models.PAYMENT_INITIATION_TYPE_PAYOUT:
		task, err := s.engine.CreatePayout(ctx, paymentInitiation.ID, 1, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	}

	return models.Task{}, nil
}
