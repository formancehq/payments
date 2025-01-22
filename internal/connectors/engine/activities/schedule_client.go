package activities

import (
	context "context"

	"go.temporal.io/sdk/client"
)

//go:generate mockgen -source schedule_client.go -destination schedule_client_generated.go -package activities . ScheduleClient ScheduleHandle
type ScheduleClient interface {
	Create(ctx context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error)
	List(ctx context.Context, options client.ScheduleListOptions) (client.ScheduleListIterator, error)
	GetHandle(ctx context.Context, scheduleID string) client.ScheduleHandle
}

type ScheduleHandle interface {
	GetID() string
	Delete(ctx context.Context) error
	Backfill(ctx context.Context, options client.ScheduleBackfillOptions) error
	Update(ctx context.Context, options client.ScheduleUpdateOptions) error
	Describe(ctx context.Context) (*client.ScheduleDescription, error)
	Trigger(ctx context.Context, options client.ScheduleTriggerOptions) error
	Pause(ctx context.Context, options client.SchedulePauseOptions) error
	Unpause(ctx context.Context, options client.ScheduleUnpauseOptions) error
}
