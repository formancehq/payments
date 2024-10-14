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

	return p, nil
}
