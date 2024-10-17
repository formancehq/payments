package workflow

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/query"
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
				// TODO(polo): context.Background() ?
				scheduleHandler := w.temporalClient.ScheduleClient().GetHandle(context.Background(), s.ID)
				if err := scheduleHandler.Delete(context.Background()); err != nil {
					// TODO(polo): log error but continue
					_ = err
				}
			})

			// TODO(polo): delete workflow execution ?
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

var RunTerminateSchedules any

func init() {
	RunTerminateSchedules = Workflow{}.runTerminateSchedules
}
