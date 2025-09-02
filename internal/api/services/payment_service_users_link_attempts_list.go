package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersLinkAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query storage.ListPSUOpenBankingConnectionAttemptsQuery) (*bunpaginate.Cursor[models.PSUOpenBankingConnectionAttempt], error) {
	_, err := s.storage.PaymentServiceUsersGet(ctx, psuID)
	if err != nil {
		return nil, newStorageError(err, "cannot get payment service user")
	}

	_, err = s.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return nil, newStorageError(err, "cannot get connector")
	}

	attempts, err := s.storage.PSUOpenBankingConnectionAttemptsList(ctx, psuID, connectorID, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment service users link attempts")
	}

	return attempts, nil
}
