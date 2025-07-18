package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersLinkAttemptsList(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, query storage.ListPSUBankBridgeConnectionAttemptsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnectionAttempt], error) {
	attempts, err := s.storage.PSUBankBridgeConnectionAttemptsList(ctx, psuID, connectorID, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list payment service users link attempts")
	}

	return attempts, nil
}
