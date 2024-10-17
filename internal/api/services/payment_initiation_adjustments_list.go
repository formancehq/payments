package services

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationAdjustmentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationAdjustment], error) {
	cursor, err := s.storage.PaymentInitiationAdjustmentsList(ctx, id, query)
	return cursor, newStorageError(err, "failed to list payment initiation adjustments")
}

func (s *Service) PaymentInitiationAdjustmentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.PaymentInitiationAdjustment, error) {
	q := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(50),
	)
	var next string
	adjustments := []models.PaymentInitiationAdjustment{}
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
