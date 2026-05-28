package services

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) OrdersList(ctx context.Context, query storage.ListOrdersQuery) (*paginate.Cursor[models.Order], error) {
	cursor, err := s.storage.OrdersList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list orders")
	}

	return cursor, nil
}
