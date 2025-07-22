package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersForward(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return handleEngineErrors(s.engine.ForwardPaymentServiceUser(ctx, psuID, connectorID))
}
