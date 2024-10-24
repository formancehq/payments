package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageEventsSentExists(ctx context.Context, id models.EventID) (bool, error) {
	isExisting, err := a.storage.EventsSentExists(ctx, id)
	if err != nil {
		return false, temporalStorageError(err)
	}
	return isExisting, nil
}

var StorageEventsSentGetActivity = Activities{}.StorageEventsSentExists

func StorageEventsSentExists(ctx workflow.Context, ik string, connectorID *models.ConnectorID) (bool, error) {
	var result bool
	err := executeActivity(ctx, StorageEventsSentGetActivity, &result, models.EventID{
		EventIdempotencyKey: ik,
		ConnectorID:         connectorID,
	})
	return result, err
}
