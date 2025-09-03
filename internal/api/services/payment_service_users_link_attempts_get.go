package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersLinkAttemptsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, id uuid.UUID) (*models.PSUOpenBankingConnectionAttempt, error) {
	_, err := s.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment service user")
	}

	_, err = s.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, newStorageError(err, "cannot get connector")
	}

	attempt, err := s.storage.PSUOpenBankingConnectionAttemptsGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment service users link attempt")
	}

	return attempt, nil
}
