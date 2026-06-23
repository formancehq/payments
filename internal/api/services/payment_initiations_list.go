package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationsList(ctx context.Context, query storage.ListPaymentInitiationsQuery) (*paginate.Cursor[models.PaymentInitiation], error) {
	pis, err := s.storage.PaymentInitiationsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment initiations")
	}

	return pis, nil
}
