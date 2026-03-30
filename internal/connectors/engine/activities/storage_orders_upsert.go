package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOrdersUpsert(ctx context.Context, orders []models.Order) error {
	return temporalStorageError(a.storage.OrdersUpsert(ctx, orders))
}

var StorageOrdersUpsertActivity = Activities{}.StorageOrdersUpsert

func StorageOrdersUpsert(ctx workflow.Context, orders []models.Order) error {
	return executeActivity(ctx, StorageOrdersUpsertActivity, nil, orders)
}
