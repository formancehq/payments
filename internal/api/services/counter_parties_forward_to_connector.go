package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) CounterPartiesForwardToConnector(ctx context.Context, counterPartyID uuid.UUID, connectorID models.ConnectorID) (models.Task, error) {
	task, err := s.engine.ForwardCounterParty(ctx, counterPartyID, connectorID)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}
	return task, nil
}
