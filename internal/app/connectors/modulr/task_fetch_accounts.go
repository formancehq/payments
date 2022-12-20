package modulr

import (
	"context"

	"github.com/formancehq/payments/internal/app/connectors/modulr/client"
	"github.com/formancehq/payments/internal/app/task"

	"github.com/formancehq/go-libs/sharedlogging"
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
				Name:      "Fetch transactions from client by account",
				Key:       taskNameFetchTransactions,
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