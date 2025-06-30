package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersCreateLink(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, idempotencyKey *uuid.UUID, ClientRedirectURL *string) (models.Task, error) {
	task, err := s.engine.CreateUserLink(ctx, psuID, connectorID, idempotencyKey, ClientRedirectURL)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
