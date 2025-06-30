package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

// TODO(polo): add tests
func (s *Service) PaymentServiceUsersUpdateLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error) {
	task, err := s.engine.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, ClientRedirectURL)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
