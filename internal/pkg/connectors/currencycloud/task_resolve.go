package currencycloud

import (
	"fmt"

	"github.com/numary/payments/internal/pkg/connectors/currencycloud/client"

	"github.com/numary/payments/internal/pkg/task"

	"github.com/numary/go-libs/sharedlogging"
)

const (
	taskNameFetchTransactions = "fetch-transactions"
)

// TaskDescriptor is the definition of a task.
type TaskDescriptor struct {
	Name string `json:"name" yaml:"name" bson:"name"`
}

func resolveTasks(logger sharedlogging.Logger, config Config) func(taskDefinition TaskDescriptor) task.Task {
	currencyCloudClient, err := client.NewClient(config.LoginID, config.APIKey, config.Endpoint)
	if err != nil {
		return func(taskDefinition TaskDescriptor) task.Task {
			return func() error {
				return fmt.Errorf("failed to initiate client: %w", err)
			}
		}
	}

	return func(taskDescriptor TaskDescriptor) task.Task {
		switch taskDescriptor.Name {
		case taskNameFetchTransactions:
			return taskFetchTransactions(logger, currencyCloudClient, config)
		}

		// This should never happen.
		return func() error {
			return fmt.Errorf("key '%s': %w", taskDescriptor.Name, ErrMissingTask)
		}
	}
}
