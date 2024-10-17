package services

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationsDelete(ctx context.Context, id models.PaymentInitiationID) error {
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
		return fmt.Errorf("cannot delete an already approved payment initiation: %w", ErrValidation)
	}

	return newStorageError(s.storage.PaymentInitiationsDelete(ctx, id), "cannot delete payment initiation")
}
