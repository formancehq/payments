package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageEventsSentStore(ctx context.Context, eventSent models.EventSent) error {
	return temporalStorageError(a.storage.EventsSentUpsert(ctx, eventSent))
}

var StorageEventsSentStoreActivity = Activities{}.StorageEventsSentStore

func StorageEventsSentStore(ctx workflow.Context, eventSent models.EventSent) error {
	return executeActivity(ctx, StorageEventsSentStoreActivity, nil, eventSent)
}
