package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentServiceUsersList(ctx context.Context, query storage.ListPSUsQuery) (*bunpaginate.Cursor[models.PaymentServiceUser], error) {
	psus, err := s.storage.PaymentServiceUsersList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment service users")
	}

	return psus, nil
}
