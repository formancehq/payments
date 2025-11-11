package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersForwardBankAccountToConnector(ctx context.Context, psuID, bankAccountID uuid.UUID, connectorID models.ConnectorID) (models.Task, error) {
	ba, err := s.storage.BankAccountsGet(ctx, bankAccountID, true)
	if err != nil {
		return models.Task{}, newStorageError(err, "failed to get bank account")
	}

	if ba == nil {
		// Should not happen, but just in case
		return models.Task{}, newStorageError(storage.ErrNotFound, "bank account not found")
	}

	psu, err := s.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		return models.Task{}, newStorageError(err, "failed to get payment service user")
	}

	models.FillBankAccountMetadataWithPaymentServiceUserInfo(ba, psu)

	// Persist enriched metadata so that subsequent outbox payloads include PSU owner information
	// TODO double check with Paul if that's OK -- up until now they were not saved as part of the bank account (only the account, once forwarded)
	if err := s.storage.BankAccountsUpdateMetadata(ctx, ba.ID, ba.Metadata); err != nil {
		return models.Task{}, newStorageError(err, "failed to update bank account metadata")
	}

	task, err := s.engine.ForwardBankAccount(ctx, *ba, connectorID, false)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
