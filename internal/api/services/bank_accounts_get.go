package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) BankAccountsGet(ctx context.Context, id uuid.UUID) (*models.BankAccount, error) {
	ba, err := s.storage.BankAccountsGet(ctx, id, true)
	if err != nil {
		return nil, newStorageError(err, "cannot get bank account")
	}

	return ba, nil
}
