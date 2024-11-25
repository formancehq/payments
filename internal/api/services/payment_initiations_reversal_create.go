package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) PaymentInitiationReversalsCreate(ctx context.Context, reversal models.PaymentInitiationReversal, waitResult bool) (models.Task, error) {
	pi, err := s.storage.PaymentInitiationsGet(ctx, reversal.PaymentInitiationID)
	if err != nil {
		return models.Task{}, newStorageError(err, "cannot create payment initiation reversal")
	}

	if err := s.storage.PaymentInitiationReversalsUpsert(
		ctx,
		reversal,
		[]models.PaymentInitiationReversalAdjustment{
			{
				ID: models.PaymentInitiationReversalAdjustmentID{
					PaymentInitiationReversalID: reversal.ID,
					CreatedAt:                   reversal.CreatedAt,
					Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING,
				},
				PaymentInitiationReversalID: reversal.ID,
				CreatedAt:                   reversal.CreatedAt,
				Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
			},
		},
	); err != nil {
		return models.Task{}, newStorageError(err, "cannot create payment initiation reversal")
	}

	switch pi.Type {
	case models.PAYMENT_INITIATION_TYPE_TRANSFER:
		task, err := s.engine.ReverseTransfer(ctx, reversal, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	case models.PAYMENT_INITIATION_TYPE_PAYOUT:
		task, err := s.engine.ReversePayout(ctx, reversal, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	}

	return models.Task{}, nil
}
