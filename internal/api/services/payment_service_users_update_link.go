package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersUpdateLink(ctx context.Context, applicationName string, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error) {
	attemptID, link, err := s.engine.UpdatePaymentServiceUserLink(ctx, applicationName, psuID, connectorID, connectionID, idempotencyKey, ClientRedirectURL)
	if err != nil {
		return "", "", handleEngineErrors(err)
	}

	return attemptID, link, nil
}
