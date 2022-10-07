package wise

import (
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

const (
	taskNameFetchTransfers = "fetch-transfers"
	taskNameFetchProfiles  = "fetch-profiles"
)

// TaskDefinition is the definition of a task.
type TaskDefinition struct {
	Name      string `json:"name" yaml:"name" bson:"name"`
	ProfileID uint64 `json:"profileID" yaml:"profileID" bson:"profileID"`
}

func resolveTasks(logger sharedlogging.Logger, config Config) func(taskDefinition TaskDefinition) task.Task {
	client := newClient(config.APIKey)

	return func(taskDefinition TaskDefinition) task.Task {
		switch taskDefinition.Name {
		case taskNameFetchProfiles:
			return taskFetchProfiles(logger, client)
		case taskNameFetchTransfers:
			return taskFetchTransfers(logger, client, taskDefinition.ProfileID)
		}

		// This should never happen.
		return func() error {
			return fmt.Errorf("key '%s': %w", taskDefinition.Name, ErrMissingTask)
		}
	}
}
