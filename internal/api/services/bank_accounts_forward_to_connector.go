package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) BankAccountsForwardToConnector(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error) {
	task, err := s.engine.ForwardBankAccount(ctx, bankAccountID, connectorID, waitResult)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}
	return task, nil
}
