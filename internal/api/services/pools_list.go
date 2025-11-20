package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"golang.org/x/sync/errgroup"
)

func (s *Service) PoolsList(ctx context.Context, query storage.ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error) {
	ps, err := s.storage.PoolsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list pools")
	}

	errGroup, egCtx := errgroup.WithContext(ctx)
	for i := range ps.Data {
		index := i
		errGroup.Go(func() error {
			if ps.Data[index].Type == models.POOL_TYPE_DYNAMIC {
				ps.Data[index].PoolAccounts, err = s.populatePoolAccounts(egCtx, &ps.Data[index])
				if err != nil {
					return newStorageError(err, "cannot populate pool accounts")
				}
			}
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return nil, newStorageError(err, "cannot populate pool accounts")
	}

	return ps, nil
}
