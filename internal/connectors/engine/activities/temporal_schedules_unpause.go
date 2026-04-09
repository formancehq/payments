package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) TemporalSchedulesUnpause(ctx context.Context, schedules []models.Schedule) error {
	for _, s := range schedules {
		handle := a.temporalClient.ScheduleClient().GetHandle(ctx, s.ID)
		if err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{}); err != nil {
			return err
		}

		if err := a.storage.SchedulesUnpause(ctx, s.ID, s.ConnectorID); err != nil {
			return err
		}
	}
	return nil
}

var TemporalSchedulesUnpauseActivity = Activities{}.TemporalSchedulesUnpause

func TemporalSchedulesUnpause(ctx workflow.Context, schedules []models.Schedule) error {
	return executeActivity(ctx, TemporalSchedulesUnpauseActivity, nil, schedules)
}
