package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type TasksTreeStoreRequest struct {
	ConnectorID models.ConnectorID
	Workflow    models.ConnectorTasksTree
}

func (a Activities) StorageConnectorTasksTreeStore(ctx context.Context, request TasksTreeStoreRequest) error {
	return temporalStorageError(a.storage.ConnectorTasksTreeUpsert(ctx, request.ConnectorID, request.Workflow))
}

var StorageConnectorTasksTreeStoreActivity = Activities{}.StorageConnectorTasksTreeStore

func StorageConnectorTasksTreeStore(ctx workflow.Context, connectorID models.ConnectorID, workflow models.ConnectorTasksTree) error {
	return executeActivity(ctx, StorageConnectorTasksTreeStoreActivity, nil, TasksTreeStoreRequest{
		ConnectorID: connectorID,
		Workflow:    workflow,
	})
}
