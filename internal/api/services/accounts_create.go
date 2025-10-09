package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) AccountsCreate(ctx context.Context, account models.Account) (*models.Account, error) {
	if err := s.engine.CreateFormanceAccount(ctx, account); err != nil {
		return nil, handleEngineErrors(err)
	}

	acc, err := s.storage.AccountsGet(ctx, account.ID)
	if err != nil {
		return nil, newStorageError(err, "cannot get account")
	}
	return acc, nil
}
