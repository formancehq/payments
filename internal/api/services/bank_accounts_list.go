package services

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

func (s *Service) BankAccountsList(ctx context.Context, query storage.ListBankAccountsQuery) (*bunpaginate.Cursor[models.BankAccount], error) {
	bas, err := s.storage.BankAccountsList(ctx, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list bank accounts")
	}

	return bas, nil
}
