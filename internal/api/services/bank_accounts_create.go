package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) BankAccountsCreate(ctx context.Context, bankAccount models.BankAccount) error {
	return newStorageError(s.storage.BankAccountsUpsert(ctx, bankAccount), "cannot create bank account")
}
