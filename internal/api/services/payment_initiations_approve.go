package services

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationsApprove(ctx context.Context, id models.PaymentInitiationID) error {
	cursor, err := s.storage.PaymentInitiationAdjustmentsList(
		ctx,
		id,
		storage.NewListPaymentInitiationAdjustmentsQuery(
			bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
				WithPageSize(1),
		),
	)
	if err != nil {
		return newStorageError(err, "cannot list payment initiation's adjustments")
	}

	if len(cursor.Data) == 0 {
		return errors.New("payment initiation's adjustments not found")
	}

	lastAdjustment := cursor.Data[0]

	if lastAdjustment.Status != models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION {
		return fmt.Errorf("cannot approve an already approved payment initiation: %w", ErrValidation)
	}

	pi, err := s.storage.PaymentInitiationsGet(ctx, id)
	if err != nil {
		return newStorageError(err, "cannot get payment initiation")
	}

	switch pi.Type {
	case models.PAYMENT_INITIATION_TYPE_TRANSFER:
		return handleEngineErrors(s.engine.CreateTransfer(ctx, pi.ID, 1))
	case models.PAYMENT_INITIATION_TYPE_PAYOUT:
		return handleEngineErrors(s.engine.CreatePayout(ctx, pi.ID, 1))
	}

	return nil
}
