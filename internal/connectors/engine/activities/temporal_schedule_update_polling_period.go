package activities

import (
	"context"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalScheduleUpdatePollingPeriod(ctx context.Context, scheduleID string, pollingPeriod time.Duration) error {
	handle := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			input.Description.Schedule.Spec.Intervals = []client.ScheduleIntervalSpec{
				{
					Every: pollingPeriod,
				},
			}
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
	if err != nil {
		return err
	}
	return nil
}

var TemporalScheduleUpdatePollingPeriodActivity = Activities{}.TemporalScheduleUpdatePollingPeriod

func TemporalScheduleUpdatePollingPeriod(ctx workflow.Context, scheduleID string, pollingPeriod time.Duration) error {
	return executeActivity(ctx, TemporalScheduleUpdatePollingPeriodActivity, nil, scheduleID, pollingPeriod)
}
