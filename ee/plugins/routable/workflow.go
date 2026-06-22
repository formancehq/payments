package routable

import "github.com/formancehq/payments/pkg/domain/models"

// workflow declares the periodic sync graph for Routable. Settings accounts
// drive balance fetches; companies and payments are independent siblings.
// All branches are periodic so the engine reschedules them under the
// connector's PollingPeriod.
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
			TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
			Name:         "fetch_external_accounts",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
