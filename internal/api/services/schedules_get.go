package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) SchedulesGet(ctx context.Context, id string, connectorID models.ConnectorID) (*models.Schedule, error) {
	schedule, err := s.storage.SchedulesGet(ctx, id, connectorID)
	if err != nil {
		return nil, newStorageError(err, "cannot get schedule")
	}

	return schedule, nil
}
