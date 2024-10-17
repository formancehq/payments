package services

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationsRetry(ctx context.Context, id models.PaymentInitiationID) error {
	adjustments, err := s.getAllPaymentInitiationAdjustments(ctx, id)
	if err != nil {
		return err
	}

	if len(adjustments) == 0 {
		return errors.New("payment initiation's adjustments not found")
	}

	lastAdjustment := adjustments[0]

	isReversed := false
	switch lastAdjustment.Status {
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED:
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED:
		isReversed = true
	default:
		return fmt.Errorf("cannot retry an already processed payment initiation: %w", ErrValidation)
	}

	pi, err := s.storage.PaymentInitiationsGet(ctx, id)
	if err != nil {
		return newStorageError(err, "cannot get payment initiation")
	}

	attempts := getAttemps(adjustments, isReversed)

	if isReversed {
		// TODO(polo): implement the reverse retry
		return fmt.Errorf("cannot retry a reversed payment initiation: %w", ErrValidation)
	} else {
		switch pi.Type {
		case models.PAYMENT_INITIATION_TYPE_TRANSFER:
			return handleEngineErrors(s.engine.CreateTransfer(ctx, pi.ID, attempts+1))
		case models.PAYMENT_INITIATION_TYPE_PAYOUT:
			return handleEngineErrors(s.engine.CreatePayout(ctx, pi.ID, attempts+1))
		}
	}

	return nil
}

func getAttemps(adjustments []models.PaymentInitiationAdjustment, isReversed bool) int {
	attempts := 0
	for _, adjustment := range adjustments {
		if isReversed && adjustment.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED {
			attempts++
		} else if !isReversed && adjustment.Status == models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED {
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
			return nil, newStorageError(err, "cannot list payment initiation's adjustments")
		}

		adjustments = append(adjustments, cursor.Data...)

		if cursor.Next == "" {
			break
		}
	}

	return adjustments, nil
}
