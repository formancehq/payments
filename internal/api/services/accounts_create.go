package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) AccountsCreate(ctx context.Context, account models.Account) error {
	return handleEngineErrors(s.engine.CreateFormanceAccount(ctx, account))
}
