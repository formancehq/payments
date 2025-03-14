package services

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) CounterPartiesList(ctx context.Context, q storage.ListCounterPartiesQuery) (*bunpaginate.Cursor[models.CounterParty], error) {
	cps, err := s.storage.CounterPartiesList(ctx, q)
	if err != nil {
		return nil, newStorageError(err, "cannot list counter parties")
	}

	return cps, nil
}
