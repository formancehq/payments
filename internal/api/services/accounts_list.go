package services

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) AccountsList(ctx context.Context, query storage.ListAccountsQuery) (*bunpaginate.Cursor[models.Account], error) {
	accounts, err := s.storage.AccountsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list accounts")
	}

	return accounts, nil
}
