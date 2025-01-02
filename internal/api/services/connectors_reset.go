package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsReset(ctx context.Context, connectorID models.ConnectorID) (models.Task, error) {
	task, err := s.engine.ResetConnector(ctx, connectorID)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}
	return task, nil
}
