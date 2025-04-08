package workflow

import (
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) runTerminateSchedules(
	ctx workflow.Context,
	uninstallConnector UninstallConnector,
) error {
	query := storage.NewListSchedulesQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ScheduleQuery{}).
			WithPageSize(100).
			WithQueryBuilder(
				query.Match("connector_id", uninstallConnector.ConnectorID.String()),
			),
	)
	for {
		schedules, err := activities.StorageSchedulesList(infiniteRetryContext(ctx), query)
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

		err = bunpaginate.UnmarshalCursor(schedules.Next, &query)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunTerminateSchedules = "TerminateSchedules"
