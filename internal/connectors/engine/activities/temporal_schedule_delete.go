package activities

import (
	"context"
	"errors"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalScheduleDelete(ctx context.Context, scheduleID string) error {
	handle := a.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		var applicationErr *temporal.ApplicationError
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &applicationErr) {
			switch applicationErr.Type() {
			case "NotFound":
				return nil
			default:
				return err
			}
		} else if errors.As(err, &notFoundErr) {
			// if the workflow is already done or doesn't exist in temporal server we can safely move on
			a.logger.Debugf("skipping deletion of schedule %q due to service error: %s", scheduleID, err.Error())
			return nil
		}
		return err
	}
	return nil
}

var TemporalScheduleDeleteActivity = Activities{}.TemporalScheduleDelete

func TemporalScheduleDelete(ctx workflow.Context, scheduleID string) error {
	return executeActivity(ctx, TemporalScheduleDeleteActivity, nil, scheduleID)
}
