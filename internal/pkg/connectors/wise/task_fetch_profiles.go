package wise

import (
	"context"
	"fmt"

	"github.com/numary/payments/internal/pkg/task"

	"github.com/numary/go-libs/sharedlogging"
)

func taskFetchProfiles(logger sharedlogging.Logger, client *client) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDescriptor],
	) error {
		profiles, err := client.getProfiles()
		if err != nil {
			return err
		}

		for _, profile := range profiles {
			logger.Infof(fmt.Sprintf("scheduling fetch-transfers: %d", profile.ID))

			def := TaskDescriptor{
				Name:      "Fetch transfers from client by profile",
				Key:       taskNameFetchTransfers,
				ProfileID: profile.ID,
			}

			err = scheduler.Schedule(def, false)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
