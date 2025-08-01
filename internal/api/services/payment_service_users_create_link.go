package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersCreateLink(ctx context.Context, clientName string, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (string, string, error) {
	attemptID, link, err := s.engine.CreatePaymentServiceUserLink(ctx, clientName, psuID, connectorID, idempotencyKey, ClientRedirectURL)
	if err != nil {
		return "", "", handleEngineErrors(err)
	}

	return attemptID, link, nil
}
