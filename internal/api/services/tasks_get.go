package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) TaskGet(ctx context.Context, id models.TaskID) (*models.Task, error) {
	task, err := s.storage.TasksGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get task")
	}

	return task, nil
}
