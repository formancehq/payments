package moov

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_OTHERS,
			Name:         "fetch_accounts",
			Periodically: true,
			TaskTreeFetchOther: &models.TaskTreeFetchOther{
				Name: "accounts",
			},
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_ACCOUNTS,
					Name:         "fetch_wallets",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
				{
					TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
					Name:         "fetch_bank_accounts",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
				{
					TaskType:     models.TASK_FETCH_PAYMENTS,
					Name:         "fetch_transfers",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		},
	}
}