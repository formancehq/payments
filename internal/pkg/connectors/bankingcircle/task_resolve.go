package bankingcircle

import (
	"fmt"

	"github.com/formancehq/payments/internal/pkg/task"

	"github.com/formancehq/go-libs/sharedlogging"
)

const (
	taskNameFetchPayments = "fetch-payments"
)

// TaskDescriptor is the definition of a task.
type TaskDescriptor struct {
	Name string `json:"name" yaml:"name" bson:"name"`
	Key  string `json:"key" yaml:"key" bson:"key"`
}

func resolveTasks(logger sharedlogging.Logger, config Config) func(taskDefinition TaskDescriptor) task.Task {
	bankingCircleClient, err := newClient(config.Username, config.Password, config.Endpoint, config.AuthorizationEndpoint, logger)
	if err != nil {
		logger.Error(err)

		return nil
	}

	return func(taskDefinition TaskDescriptor) task.Task {
		switch taskDefinition.Key {
		case taskNameFetchPayments:
			return taskFetchPayments(logger, bankingCircleClient)
		}

		// This should never happen.
		return func() error {
			return fmt.Errorf("key '%s': %w", taskDefinition.Key, ErrMissingTask)
		}
	}
}
