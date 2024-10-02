package services

import (
	"context"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) PoolsList(ctx context.Context, query storage.ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error) {
	ps, err := s.storage.PoolsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list pools")
	}

	return ps, nil
}
