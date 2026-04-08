package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalSchedulesPause(ctx context.Context, instances []models.Instance) error {
	now := time.Now()
	for _, instance := range instances {
		reason := ""
		if instance.Error != nil {
			reason = *instance.Error
		}

		handle := a.temporalClient.ScheduleClient().GetHandle(ctx, instance.ScheduleID)
		if err := handle.Pause(ctx, client.SchedulePauseOptions{
			Note: reason,
		}); err != nil {
			return err
		}

		if err := a.storage.SchedulesPause(ctx, instance.ScheduleID, now, reason); err != nil {
			return err
		}
	}
	return nil
}

var TemporalSchedulesPauseActivity = Activities{}.TemporalSchedulesPause

func TemporalSchedulesPause(ctx workflow.Context, instances []models.Instance) error {
	return executeActivity(ctx, TemporalSchedulesPauseActivity, nil, instances)
}
