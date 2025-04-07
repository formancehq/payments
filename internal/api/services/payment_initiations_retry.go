package services

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationsRetry(ctx context.Context, id models.PaymentInitiationID, waitResult bool) (models.Task, error) {
	adjustments, err := s.getAllPaymentInitiationAdjustments(ctx, id)
	if err != nil {
		return models.Task{}, err
	}

	if len(adjustments) == 0 {
		return models.Task{}, errors.New("payment initiation adjustments not found")
	}

	lastAdjustment := adjustments[0]

	switch lastAdjustment.Status {
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED:
	default:
		return models.Task{}, fmt.Errorf("cannot retry an already processed payment initiation: %w", ErrValidation)
	}

	pi, err := s.storage.PaymentInitiationsGet(ctx, id)
	if err != nil {
		return models.Task{}, newStorageError(err, "cannot get payment initiation")
	}

	attempts := getAttemps(adjustments)

	switch pi.Type {
	case models.PAYMENT_INITIATION_TYPE_TRANSFER:
		task, err := s.engine.CreateTransfer(ctx, pi.ID, attempts+1, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	case models.PAYMENT_INITIATION_TYPE_PAYOUT:
		task, err := s.engine.CreatePayout(ctx, pi.ID, attempts+1, waitResult)
		if err != nil {
			return models.Task{}, handleEngineErrors(err)
		}
		return task, nil
	}

	return models.Task{}, nil
}

func getAttemps(adjustments []models.PaymentInitiationAdjustment) int {
	attempts := 0
	for _, adjustment := range adjustments {
		if adjustment.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED {
			attempts++
		}
	}

	return attempts
}

func (s *Service) getAllPaymentInitiationAdjustments(ctx context.Context, id models.PaymentInitiationID) ([]models.PaymentInitiationAdjustment, error) {
	adjustments := []models.PaymentInitiationAdjustment{}
	q := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(50),
	)
	var next string
	for {
		if next != "" {
			err := bunpaginate.UnmarshalCursor(next, &q)
			if err != nil {
				return nil, err
			}
		}

		cursor, err := s.storage.PaymentInitiationAdjustmentsList(ctx, id, q)
		if err != nil {
			return nil, newStorageError(err, "cannot list payment initiation adjustments")
		}

		adjustments = append(adjustments, cursor.Data...)

		if cursor.Next == "" {
			break
		}
	}

	return adjustments, nil
}
