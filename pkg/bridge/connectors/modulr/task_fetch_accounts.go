package modulr

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/connectors/modulr/client"
	"github.com/numary/payments/pkg/bridge/task"
)

func taskFetchAccounts(logger sharedlogging.Logger, client *client.Client) task.Task {
	return func(
		ctx context.Context,
		scheduler task.Scheduler[TaskDescriptor],
	) error {
		logger.Info(taskNameFetchAccounts)

		accounts, err := client.GetAccounts()
		if err != nil {
			return err
		}

		for _, account := range accounts {
			logger.Infof("scheduling fetch-transactions: %s", account.ID)

			transactionsTask := TaskDescriptor{
				Name:      taskNameFetchTransactions,
				AccountID: account.ID,
			}

			err = scheduler.Schedule(transactionsTask, false)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
