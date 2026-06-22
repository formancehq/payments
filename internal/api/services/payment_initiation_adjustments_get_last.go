package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

func (s *Service) PaymentInitiationAdjustmentsGetLast(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiationAdjustment, error) {
	q := storage.NewListPaymentInitiationAdjustmentsQuery(
		paginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(1),
	)

	cursor, err := s.storage.PaymentInitiationAdjustmentsList(ctx, id, q)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment initiation's adjustments")
	}

	if len(cursor.Data) == 0 {
		return nil, errors.New("payment initiation's adjustments not found")
	}

	return &cursor.Data[0], nil
}
