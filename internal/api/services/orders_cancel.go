package services

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (s *Service) OrdersCancel(ctx context.Context, id models.OrderID) error {
	// First, get the order to check its current status
	order, err := s.storage.OrdersGet(ctx, id)
	if err != nil {
		return newStorageError(err, "cannot get order")
	}

	// Check if the order can be cancelled
	if !order.Status.CanCancel() {
		return errorsutils.NewWrappedError(
			fmt.Errorf("order cannot be cancelled in status %s", order.Status.String()),
			ErrValidation,
		)
	}

	// Update the order status to CANCELLED
	err = s.storage.OrdersUpdateStatus(ctx, id, models.ORDER_STATUS_CANCELLED)
	if err != nil {
		return newStorageError(err, "cannot cancel order")
	}

	return nil
}
