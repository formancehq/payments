package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/activity"
)

// batchDeleteFunc is a function signature for batch delete operations on storage.
type batchDeleteFunc func(ctx context.Context, connectorID models.ConnectorID, batchSize int) (int, error)

// batchDeleteWithHeartbeat performs batch deletions with Temporal heartbeat reporting.
// It loops until no more rows are deleted, sending heartbeat updates with progress.
func (a Activities) batchDeleteWithHeartbeat(ctx context.Context, connectorID models.ConnectorID, deleteFunc batchDeleteFunc, statusMsg string) error {
	const batchSize = 1000
	totalDeleted := 0

	for {
		// Delete one batch and get number of rows affected
		rowsAffected, err := deleteFunc(ctx, connectorID, batchSize)
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
			"status":  statusMsg,
		})
	}

	return nil
}
