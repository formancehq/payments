package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) CounterPartiesCreate(ctx context.Context, counterParty models.CounterParty, bankAccount *models.BankAccount) error {
	return handleEngineErrors(s.engine.CreateCounterParty(ctx, counterParty, bankAccount))
}
