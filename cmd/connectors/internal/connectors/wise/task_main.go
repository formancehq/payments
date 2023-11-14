package wise

import (
	"context"
	"errors"

	"github.com/formancehq/payments/cmd/connectors/internal/task"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/logging"
)

// taskMain is the main task of the connector. It launches the other tasks.
func taskMain() task.Task {
	return func(
		ctx context.Context,
		logger logging.Logger,
		scheduler task.Scheduler,
	) error {
		logger.Info(taskNameMain)

		taskUsers, err := models.EncodeTaskDescriptor(TaskDescriptor{
			Name: "Fetch users from client",
			Key:  taskNameFetchProfiles,
		})
		if err != nil {
			return err
		}

		err = scheduler.Schedule(ctx, taskUsers, models.TaskSchedulerOptions{
			ScheduleOption: models.OPTIONS_RUN_NOW,
			RestartOption:  models.OPTIONS_RESTART_IF_NOT_ACTIVE,
		})
		if err != nil && !errors.Is(err, task.ErrAlreadyScheduled) {
			return err
		}

		return nil
	}
}
