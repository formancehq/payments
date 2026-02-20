package adyen

import "github.com/formancehq/payments/pkg/connector"

func workflow() connector.ConnectorTasksTree {
	return []connector.ConnectorTaskTree{
		{
			TaskType:     connector.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks:    []connector.ConnectorTaskTree{},
		},
		{
			TaskType:     connector.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []connector.ConnectorTaskTree{},
		},
	}
}
