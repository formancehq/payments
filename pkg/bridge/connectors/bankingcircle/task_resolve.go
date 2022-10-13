package bankingcircle

import (
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/task"
)

const (
	taskNameFetchPayments = "fetch-payments"
)

// TaskDefinition is the definition of a task.
type TaskDefinition struct {
	Name string `json:"name" yaml:"name" bson:"name"`
}

func resolveTasks(logger sharedlogging.Logger, config Config) func(taskDefinition TaskDefinition) task.Task {
	client, err := newClient(config.Username, config.Password, config.Endpoint, config.AuthorizationEndpoint, logger)
	if err != nil {
		logger.Error(err)

		return nil
	}

	return func(taskDefinition TaskDefinition) task.Task {
		switch taskDefinition.Name {
		case taskNameFetchPayments:
			return taskFetchPayments(logger, client)
		}

		// This should never happen.
		return func() error {
			return fmt.Errorf("key '%s': %w", taskDefinition.Name, ErrMissingTask)
		}
	}
}
