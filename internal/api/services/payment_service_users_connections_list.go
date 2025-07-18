package services

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersConnectionsList(ctx context.Context, psuID uuid.UUID, connectorID *models.ConnectorID, query storage.ListPsuBankBridgeConnectionsQuery) (*bunpaginate.Cursor[models.PSUBankBridgeConnection], error) {
	ps, err := s.storage.PSUBankBridgeConnectionsList(ctx, psuID, connectorID, query)
	if err != nil {
		return nil, newStorageError(err, "cannot list psu bank bridge connections")
	}

	return ps, nil
}
