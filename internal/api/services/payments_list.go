package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentsList(ctx context.Context, query storage.ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	ps, err := s.storage.PaymentsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payments")
	}

	return ps, nil
}
