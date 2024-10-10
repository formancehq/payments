package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error) {
	account, err := s.storage.AccountsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get account")
	}

	return account, nil
}
