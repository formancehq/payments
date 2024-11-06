package activities

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalDeleteSchedule(ctx context.Context, scheduleID string) error {
	handle := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	return handle.Delete(ctx)
}

var TemporalDeleteScheduleActivity = Activities{}.TemporalDeleteSchedule

func TemporalDeleteSchedule(ctx workflow.Context, scheduleID string) error {
	return executeActivity(ctx, TemporalDeleteScheduleActivity, nil, scheduleID)
}
