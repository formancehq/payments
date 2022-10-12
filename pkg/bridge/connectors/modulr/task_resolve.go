package modulr

import (
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/connectors/modulr/client"
	"github.com/numary/payments/pkg/bridge/task"
)

const (
	taskNameFetchTransactions = "fetch-transactions"
	taskNameFetchAccounts     = "fetch-accounts"
)

// TaskDescriptor is the definition of a task.
type TaskDescriptor struct {
	Name      string `json:"name" yaml:"name" bson:"name"`
	AccountID string `json:"accountID" yaml:"accountID" bson:"accountID"`
}

func resolveTasks(logger sharedlogging.Logger, config Config) func(taskDefinition TaskDescriptor) task.Task {
	modulrClient, err := client.NewClient(config.APIKey, config.APISecret, config.Endpoint)
	if err != nil {
		return func(taskDefinition TaskDescriptor) task.Task {
			return func() error {
				return fmt.Errorf("key '%s': %w", taskDefinition.Name, ErrMissingTask)
			}
		}
	}

	return func(taskDefinition TaskDescriptor) task.Task {
		switch taskDefinition.Name {
		case taskNameFetchAccounts:
			return taskFetchAccounts(logger, modulrClient)
		case taskNameFetchTransactions:
			return taskFetchTransactions(logger, modulrClient, taskDefinition.AccountID)
		}

		// This should never happen.
		return func() error {
			return fmt.Errorf("key '%s': %w", taskDefinition.Name, ErrMissingTask)
		}
	}
}
