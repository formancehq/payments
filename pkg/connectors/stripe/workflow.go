package stripe

import "github.com/formancehq/payments/pkg/connector"

func workflow() connector.ConnectorTasksTree {
	return []connector.ConnectorTaskTree{
		{
			TaskType:     connector.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []connector.ConnectorTaskTree{
				{
					TaskType:     connector.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: false,
					NextTasks:    []connector.ConnectorTaskTree{},
				},
				{
					TaskType:     connector.TASK_FETCH_PAYMENTS,
					Name:         "fetch_payments",
					Periodically: true,
					NextTasks:    []connector.ConnectorTaskTree{},
				},
				{
					TaskType:     connector.TASK_FETCH_EXTERNAL_ACCOUNTS,
					Name:         "fetch_recipients",
					Periodically: true,
					NextTasks:    []connector.ConnectorTaskTree{},
				},
			},
		},
		{
			TaskType:     connector.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []connector.ConnectorTaskTree{},
		},
	}
}
