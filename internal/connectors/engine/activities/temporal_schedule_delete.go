package activities

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalScheduleDelete(ctx context.Context, scheduleID string) error {
	handle := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) {
			switch applicationErr.Type() {
			case "NotFound":
				return nil
			default:
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

var TemporalScheduleDeleteActivity = Activities{}.TemporalScheduleDelete

func TemporalScheduleDelete(ctx workflow.Context, scheduleID string) error {
	return executeActivity(ctx, TemporalScheduleDeleteActivity, nil, scheduleID)
}
