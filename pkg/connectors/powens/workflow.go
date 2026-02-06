package powens

import "github.com/formancehq/payments/pkg/connector"

func workflow() connector.ConnectorTasksTree {
	return []connector.ConnectorTaskTree{
		{
			TaskType:     connector.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []connector.ConnectorTaskTree{},
		},
	}
}
