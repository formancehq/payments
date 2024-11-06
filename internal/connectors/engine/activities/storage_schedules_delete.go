package activities

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageSchedulesDelete(ctx context.Context, scheduleID string) error {
	return a.storage.SchedulesDelete(ctx, scheduleID)
}

var StorageSchedulesDeleteActivity = Activities{}.StorageSchedulesDelete

func StorageSchedulesDelete(ctx workflow.Context, scheduleID string) error {
	return executeActivity(ctx, StorageSchedulesDeleteActivity, nil, scheduleID)
}
