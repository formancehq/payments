package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageStatesStore(ctx context.Context, state models.State) error {
	return temporalStorageError(a.storage.StatesUpsert(ctx, state))
}

var StorageStatesStoreActivity = Activities{}.StorageStatesStore

func StorageStatesStore(ctx workflow.Context, state models.State) error {
	return executeActivity(ctx, StorageStatesStoreActivity, nil, state)
}
