package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageSchedulesGet(ctx context.Context, connectorID models.ConnectorID, scheduleID string) (*models.Schedule, error) {
	resp, err := a.storage.SchedulesGet(ctx, scheduleID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return resp, nil
}

var StorageSchedulesGetActivity = Activities{}.StorageSchedulesGet

func StorageSchedulesGet(ctx workflow.Context, connectorID models.ConnectorID, scheduleID string) (*models.Schedule, error) {
	ret := models.Schedule{}
	if err := executeActivity(ctx, StorageSchedulesGetActivity, &ret, connectorID, scheduleID); err != nil {
		return nil, err
	}
	return &ret, nil
}
