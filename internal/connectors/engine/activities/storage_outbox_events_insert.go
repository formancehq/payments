package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOutboxEventsInsert(ctx context.Context, events []models.OutboxEvent) error {
	return temporalStorageError(a.storage.OutboxEventsInsertWithTx(ctx, events))
}

var StorageOutboxEventsInsertActivity = Activities{}.StorageOutboxEventsInsert

// StorageOutboxEventsInsert is meant to be used only when there's no related entity written to DB: if there's a DB
// entity, the event should be created within the same transaction as the entity.
func StorageOutboxEventsInsert(ctx workflow.Context, events []models.OutboxEvent) error {
	return executeActivity(ctx, StorageOutboxEventsInsertActivity, nil, events)
}
