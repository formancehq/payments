package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageTradesStore(ctx context.Context, trades []models.Trade) error {
	return temporalStorageError(a.storage.TradesUpsert(ctx, trades))
}

var StorageTradesStoreActivity = Activities{}.StorageTradesStore

func StorageTradesStore(ctx workflow.Context, trades []models.Trade) error {
	return executeActivity(ctx, StorageTradesStoreActivity, nil, trades)
}

