package dummypay

import "github.com/formancehq/payments/internal/models"

//nolint:unused
func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_TRADES,
			Name:         "fetch_trades",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
