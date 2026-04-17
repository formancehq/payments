package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) OrdersGet(ctx context.Context, id models.OrderID) (*models.Order, error) {
	order, err := s.storage.OrdersGet(ctx, id)
	if err != nil {
		return nil, newStorageError(err, "cannot get order")
	}

	return order, nil
}
