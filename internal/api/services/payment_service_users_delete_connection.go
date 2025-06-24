package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersDeleteConnection(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID, connectionID string) (models.Task, error) {
	task, err := s.engine.DeleteUserConnection(ctx, connectorID, psuID, connectionID)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
