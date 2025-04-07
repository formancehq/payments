package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageSchedulesList(ctx context.Context, query storage.ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error) {
	cursor, err := a.storage.SchedulesList(ctx, query)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

var StorageSchedulesListActivity = Activities{}.StorageSchedulesList

func StorageSchedulesList(ctx workflow.Context, query storage.ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error) {
	ret := bunpaginate.Cursor[models.Schedule]{}
	if err := executeActivity(ctx, StorageSchedulesListActivity, &ret, query); err != nil {
		return nil, err
	}
	return &ret, nil
}
