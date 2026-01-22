package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

// OrdersCreate creates an order and optionally sends it to the exchange.
// If sendToExchange is true, the order will be sent via a Temporal workflow.
// If waitResult is true, the function will wait for the workflow to complete.
func (s *Service) OrdersCreate(ctx context.Context, order models.Order, sendToExchange bool, waitResult bool) (models.Task, error) {
	// Store the order first
	err := s.storage.OrdersUpsert(ctx, []models.Order{order})
	if err != nil {
		return models.Task{}, newStorageError(err, "cannot create order")
	}

	if !sendToExchange {
		return models.Task{}, nil
	}

	// Trigger workflow to send order to exchange
	task, err := s.engine.CreateOrder(ctx, order.ID, waitResult)
	if err != nil {
		return models.Task{}, err
	}

	return task, nil
}
