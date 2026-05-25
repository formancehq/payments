package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PaymentServiceUsersList(ctx context.Context, query storage.ListPSUsQuery) (*paginate.Cursor[models.PaymentServiceUser], error) {
	psus, err := s.storage.PaymentServiceUsersList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment service users")
	}

	return psus, nil
}
