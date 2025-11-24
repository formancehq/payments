package workflow

import (
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

type UpdateSchedulePollingPeriod struct {
	ConnectorID   models.ConnectorID
	Config        models.Config
	NextPageToken string
}

func (w Workflow) runUpdateSchedulePollingPeriod(
	ctx workflow.Context,
	in UpdateSchedulePollingPeriod,
) error {
	var q storage.ListSchedulesQuery
	if in.NextPageToken != "" {
		err := bunpaginate.UnmarshalCursor(in.NextPageToken, &q)
		if err != nil {
			return err
		}
	} else {
		q = storage.NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(storage.ScheduleQuery{}).
				WithPageSize(100).
				WithQueryBuilder(
					query.Match("connector_id", in.ConnectorID.String()),
				),
		)
	}

	for {
		schedules, err := activities.StorageSchedulesList(infiniteRetryContext(ctx), q)
		if err != nil {
			return err
		}

		wg := workflow.NewWaitGroup(ctx)

		for _, schedule := range schedules.Data {
			s := schedule
			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				if err := activities.TemporalScheduleUpdatePollingPeriod(
					infiniteRetryContext(ctx),
					s.ID,
					in.Config.PollingPeriod,
				); err != nil {
					workflow.GetLogger(ctx).Error("failed to update schedule polling period", "schedule_id", s.ID, "error", err)
				}
			})
		}

		wg.Wait(ctx)

		if !schedules.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(schedules.Next, &q)
		if err != nil {
			return err
		}

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of schedules, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunUpdateSchedulePollingPeriod,
				UpdateSchedulePollingPeriod{
					ConnectorID:   in.ConnectorID,
					Config:        in.Config,
					NextPageToken: schedules.Next,
				},
			)
		}
	}

	return nil
}

const RunUpdateSchedulePollingPeriod = "UpdateSchedulePollingPeriod"
