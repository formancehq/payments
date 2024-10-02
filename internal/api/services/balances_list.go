package services

import (
	"context"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) BalancesList(ctx context.Context, query storage.ListBalancesQuery) (*bunpaginate.Cursor[models.Balance], error) {
	balances, err := s.storage.BalancesList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list balances")
	}

	return balances, nil
}
