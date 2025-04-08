package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationRelatedPaymentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	cursor, err := s.storage.PaymentInitiationRelatedPaymentsList(ctx, id, query)
	return cursor, newStorageError(err, "cannot list payment initiation related payments")
}

func (s *Service) PaymentInitiationRelatedPaymentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.Payment, error) {
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
			return nil, newStorageError(err, "cannot list payment initiation related payments")
		}

		relatedPayment = append(relatedPayment, cursor.Data...)

		if cursor.Next == "" {
			break
		}
	}

	return relatedPayment, nil
}
