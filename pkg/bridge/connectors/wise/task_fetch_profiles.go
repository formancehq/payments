package wise

import (
	"context"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

func taskFetchProfiles(logger sharedlogging.Logger, config Config) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDefinition],
	) error {
		client := newClient(config.APIKey)

		profiles, err := client.getProfiles()
		if err != nil {
			return err
		}

		for _, profile := range profiles {
			logger.Infof(fmt.Sprintf("scheduling fetch-transfers: %d", profile.ID))

			def := TaskDefinition{
				Name:      taskNameFetchTransfers,
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
