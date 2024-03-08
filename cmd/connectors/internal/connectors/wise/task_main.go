package wise

import (
	"context"

	"github.com/formancehq/payments/cmd/connectors/internal/connectors"
	"github.com/formancehq/payments/cmd/connectors/internal/task"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
)

// taskMain is the main task of the connector. It launches the other tasks.
func taskMain() task.Task {
	return func(
		ctx context.Context,
		taskID models.TaskID,
		connectorID models.ConnectorID,
		scheduler task.Scheduler,
	) error {
		ctx, span := connectors.StartSpan(
			ctx,
			"wise.taskMain",
			attribute.String("connectorID", connectorID.String()),
			attribute.String("taskID", taskID.String()),
		)
		defer span.End()

		taskUsers, err := models.EncodeTaskDescriptor(TaskDescriptor{
			Name: "Fetch users from client",
			Key:  taskNameFetchProfiles,
		})
		if err != nil {
			otel.RecordError(span, err)
			return errors.Wrap(task.ErrRetryable, err.Error())
		}

		err = scheduler.Schedule(ctx, taskUsers, models.TaskSchedulerOptions{
			ScheduleOption: models.OPTIONS_RUN_NOW,
			RestartOption:  models.OPTIONS_RESTART_IF_NOT_ACTIVE,
		})
		if err != nil && !errors.Is(err, task.ErrAlreadyScheduled) {
			otel.RecordError(span, err)
			return errors.Wrap(task.ErrRetryable, err.Error())
		}

		return nil
	}
}
