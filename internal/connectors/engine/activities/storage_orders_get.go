package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOrdersGet(ctx context.Context, id models.OrderID) (*models.Order, error) {
	order, err := a.storage.OrdersGet(ctx, id)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return order, nil
}

var StorageOrdersGetActivity = Activities{}.StorageOrdersGet

func StorageOrdersGet(ctx workflow.Context, id models.OrderID) (*models.Order, error) {
	ret := models.Order{}
	if err := executeActivity(ctx, StorageOrdersGetActivity, &ret, id); err != nil {
		return nil, err
	}
	return &ret, nil
}
