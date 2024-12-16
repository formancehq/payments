package currencycloud

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
			Name:         "fetch_beneficiaries",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{

			TaskType:     models.TASK_FETCH_BALANCES,
			Name:         "fetch_balances",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
