package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) BankAccountsForwardToConnector(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID) (*models.BankAccount, error) {
	ba, err := s.engine.ForwardBankAccount(ctx, bankAccountID, connectorID)
	if err != nil {
		return nil, handleEngineErrors(err)
	}

	return ba, nil
}
