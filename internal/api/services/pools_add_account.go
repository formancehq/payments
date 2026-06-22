package services

import (
	"context"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
)

func (s *Service) PoolsAddAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	err := s.engine.AddAccountToPool(ctx, id, accountID)
	return handleEngineErrors(err)
}
