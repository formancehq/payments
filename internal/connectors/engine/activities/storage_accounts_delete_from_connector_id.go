package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	const batchSize = 1000
	totalDeleted := 0

	for {
		// Delete one batch and get number of rows affected
		rowsAffected, err := a.storage.AccountsDeleteFromConnectorIDBatch(ctx, connectorID, batchSize)
		if err != nil {
			return temporalStorageError(err)
		}

		// If no rows were deleted, we're done
		if rowsAffected == 0 {
			break
		}

		totalDeleted += rowsAffected

		// Send heartbeat to Temporal with progress
		activity.RecordHeartbeat(ctx, map[string]interface{}{
			"deleted": totalDeleted,
			"status":  "deleting accounts",
		})
	}

	return nil
}

var StorageAccountsDeleteFromConnectorIDActivity = Activities{}.StorageAccountsDeleteFromConnectorID

func StorageAccountsDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageAccountsDeleteFromConnectorIDActivity, nil, connectorID)
}
