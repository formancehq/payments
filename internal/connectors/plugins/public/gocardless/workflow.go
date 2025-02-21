package gocardless

import "github.com/formancehq/payments/internal/models"

const (
	fetchOthers       = "fetch_others"
	fetchCustomers    = "fetch_accounts"
	fetchNextAccounts = "fetch_external_accounts"
)

func Workflow() models.ConnectorTasksTree {

	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_OTHERS,
			Name:         fetchOthers,
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_ACCOUNTS,
					Name:         fetchCustomers,
					Periodically: true,
				},
				{
					TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
					Name:         fetchNextAccounts,
					Periodically: true,
				},
			},
		},
	}
}
