package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PaymentServiceUsersDelete(ctx context.Context, psuID uuid.UUID) (models.Task, error) {
	task, err := s.engine.DeletePaymentServiceUser(ctx, psuID)
	if err != nil {
		return models.Task{}, handleEngineErrors(err)
	}

	return task, nil
}
