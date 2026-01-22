package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type OrdersUpdateStatusRequest struct {
	ID     models.OrderID
	Status models.OrderStatus
}

func (a Activities) StorageOrdersUpdateStatus(ctx context.Context, req OrdersUpdateStatusRequest) error {
	return temporalStorageError(a.storage.OrdersUpdateStatus(ctx, req.ID, req.Status))
}

var StorageOrdersUpdateStatusActivity = Activities{}.StorageOrdersUpdateStatus

func StorageOrdersUpdateStatus(ctx workflow.Context, id models.OrderID, status models.OrderStatus) error {
	return executeActivity(ctx, StorageOrdersUpdateStatusActivity, nil, OrdersUpdateStatusRequest{
		ID:     id,
		Status: status,
	})
}
