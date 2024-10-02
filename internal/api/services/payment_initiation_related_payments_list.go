package services

import (
	"context"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationRelatedPaymentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	cursor, err := s.storage.PaymentInitiationRelatedPaymentsList(ctx, id, query)
	return cursor, newStorageError(err, "failed to list payment initiation related payments")
}

func (s *Service) PaymentInitiationRelatedPaymentListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.Payment, error) {
	q := storage.NewListPaymentInitiationRelatedPaymentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationRelatedPaymentsQuery{}).
			WithPageSize(50),
	)
	var next string
	relatedPayment := []models.Payment{}
	for {
		if next != "" {
			err := bunpaginate.UnmarshalCursor(next, &q)
			if err != nil {
				return nil, err
			}
		}

		cursor, err := s.storage.PaymentInitiationRelatedPaymentsList(ctx, id, q)
		if err != nil {
			return nil, newStorageError(err, "cannot list payment initiation's adjustments")
		}

		relatedPayment = append(relatedPayment, cursor.Data...)

		if cursor.Next == "" {
			break
		}
	}

	return relatedPayment, nil
}
