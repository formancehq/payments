package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationRelatedPaymentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationRelatedPaymentsQuery) (*paginate.Cursor[models.Payment], error) {
	cursor, err := s.storage.PaymentInitiationRelatedPaymentsList(ctx, id, query)
	return cursor, newStorageError(err, "cannot list payment initiation related payments")
}

func (s *Service) PaymentInitiationRelatedPaymentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.Payment, error) {
	q := storage.NewListPaymentInitiationRelatedPaymentsQuery(
		paginate.NewPaginatedQueryOptions(storage.PaymentInitiationRelatedPaymentsQuery{}).
			WithPageSize(50),
	)
	var next string
	relatedPayment := []models.Payment{}
	for {
		if next != "" {
			err := paginate.UnmarshalCursor(next, &q)
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
