package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageTasksStore(ctx context.Context, task models.Task) error {
	return temporalStorageError(a.storage.TasksUpsert(ctx, task))
}

var StorageTasksStoreActivity = Activities{}.StorageTasksStore

func StorageTasksStore(ctx workflow.Context, task models.Task) error {
	return executeActivity(ctx, StorageTasksStoreActivity, nil, task)
}
