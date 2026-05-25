package workflow

import (
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/go-libs/v5/pkg/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

type ListActiveSchedules struct {
	ConnectorID   models.ConnectorID
	NextPageToken string
}

type ListActiveSchedulesResult struct {
	Data          []models.Schedule
	NextPageToken string
}

func (w Workflow) runListActiveSchedules(
	ctx workflow.Context,
	listActiveSchedules ListActiveSchedules,
) (result ListActiveSchedulesResult, _ error) {
	var q storage.ListSchedulesQuery
	if listActiveSchedules.NextPageToken != "" {
		err := paginate.UnmarshalCursor(listActiveSchedules.NextPageToken, &q)
		if err != nil {
			return result, err
		}
	} else {
		q = storage.NewListSchedulesQuery(
			paginate.NewPaginatedQueryOptions(storage.ScheduleQuery{}).
				WithPageSize(100).
				WithQueryBuilder(
					query.Match("connector_id", listActiveSchedules.ConnectorID.String()),
				),
		)
	}

	schedules, err := activities.StorageSchedulesList(infiniteRetryContext(ctx), q)
	if err != nil {
		return result, err
	}

	result.NextPageToken = schedules.Next
	result.Data = append(result.Data, schedules.Data...)
	return result, nil
}

const RunListActiveSchedules = "ListActiveSchedules"
