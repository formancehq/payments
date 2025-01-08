package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsUninstall(ctx context.Context, connectorID models.ConnectorID) (models.Task, error) {
	_, err := s.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return models.Task{}, newStorageError(err, "cannot get connector")
	}

	task, err := s.engine.UninstallConnector(ctx, connectorID)
	return task, handleEngineErrors(err)
}
