package activities

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalScheduleDelete(ctx context.Context, scheduleID string) error {
	handle := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	return handle.Delete(ctx)
}

var TemporalScheduleDeleteActivity = Activities{}.TemporalScheduleDelete

func TemporalScheduleDelete(ctx workflow.Context, scheduleID string) error {
	return executeActivity(ctx, TemporalScheduleDeleteActivity, nil, scheduleID)
}
