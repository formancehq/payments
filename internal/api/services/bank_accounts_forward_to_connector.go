package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) BankAccountsForwardToConnector(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error) {
	ba, err := s.storage.BankAccountsGet(ctx, bankAccountID, true)
	if err != nil {
		return models.Task{}, newStorageError(err, "failed to get bank account")
	}

	if ba == nil {
		// Should not happened, but just in case
		return models.Task{}, newStorageError(nil, "bank account not found")
	}

	task, err := s.engine.ForwardBankAccount(ctx, *ba, connectorID, waitResult)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}
	return task, nil
}
