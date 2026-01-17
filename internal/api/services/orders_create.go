package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) OrdersCreate(ctx context.Context, order models.Order) error {
	// Store the order
	err := s.storage.OrdersUpsert(ctx, []models.Order{order})
	if err != nil {
		return newStorageError(err, "cannot create order")
	}

	return nil
}
