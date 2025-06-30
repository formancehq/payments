package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersForward(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (models.Task, error) {
	task, err := s.engine.ForwardUser(ctx, psuID, connectorID)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
