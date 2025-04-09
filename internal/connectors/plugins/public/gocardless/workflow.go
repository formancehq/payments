package gocardless

import "github.com/formancehq/payments/internal/models"

const (
	fetchOthers       = "fetch_others"
	fetchNextAccounts = "fetch_external_accounts"
	fetchNextPayments = "fetch_payments"
)

func Workflow() models.ConnectorTasksTree {

	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_OTHERS,
			Name:         fetchOthers,
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
					Name:         fetchNextAccounts,
					Periodically: true,
				},
			},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         fetchNextPayments,
			Periodically: true,
		},
	}
}
