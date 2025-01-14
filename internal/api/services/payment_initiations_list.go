package services

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentInitiationsList(ctx context.Context, query storage.ListPaymentInitiationsQuery) (*bunpaginate.Cursor[models.PaymentInitiation], error) {
	pis, err := s.storage.PaymentInitiationsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment initiations")
	}

	return pis, nil
}
