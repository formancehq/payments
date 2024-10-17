package services

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationsReject(ctx context.Context, id models.PaymentInitiationID) error {
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
		return fmt.Errorf("cannot reject an already approved payment initiation: %w", ErrValidation)
	}

	now := time.Now().UTC()
	return newStorageError(s.storage.PaymentInitiationAdjustmentsUpsert(
		ctx,
		models.PaymentInitiationAdjustment{
			ID: models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: id,
				CreatedAt:           now,
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED,
			},
			PaymentInitiationID: id,
			CreatedAt:           now,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED,
		},
	), "cannot reject payment initiation")
}
