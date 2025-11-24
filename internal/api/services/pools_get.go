package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PoolsGet(ctx context.Context, id uuid.UUID) (*models.Pool, error) {
	p, err := s.storage.PoolsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get pool")
	}

	if p.Type == models.POOL_TYPE_DYNAMIC {
		// populate the pool accounts from the query
		p.PoolAccounts, err = s.populatePoolAccounts(ctx, p)
		if err != nil {
			return nil, newStorageError(err, "cannot populate pool accounts")
		}
	}

	return p, nil
}
