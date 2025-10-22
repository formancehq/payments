package workflow

import (
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

type TerminateSchedules struct {
	ConnectorID   models.ConnectorID
	NextPageToken string
}

func (w Workflow) runTerminateSchedules(
	ctx workflow.Context,
	terminateSchedules TerminateSchedules,
) error {
	var q storage.ListSchedulesQuery
	if terminateSchedules.NextPageToken != "" {
		err := bunpaginate.UnmarshalCursor(terminateSchedules.NextPageToken, &q)
		if err != nil {
			return err
		}
	} else {
		q = storage.NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(storage.ScheduleQuery{}).
				WithPageSize(100).
				WithQueryBuilder(
					query.Match("connector_id", terminateSchedules.ConnectorID.String()),
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

				if err := activities.TemporalScheduleDelete(
					infiniteRetryContext(ctx),
					s.ID,
				); err != nil {
					workflow.GetLogger(ctx).Error("failed to delete schedule", "schedule_id", s.ID, "error", err)
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
			// If we have lots and lots of accounts, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunTerminateSchedules,
				TerminateSchedules{
					ConnectorID:   terminateSchedules.ConnectorID,
					NextPageToken: schedules.Next,
				},
			)
		}
	}

	return nil
}

const RunTerminateSchedules = "TerminateSchedules"
