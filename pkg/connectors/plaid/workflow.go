package plaid

import "github.com/formancehq/payments/pkg/connector"

func workflow() connector.ConnectorTasksTree {
	// Do not launch fetch data workflows here, since we're depending on the
	// users to finish the link flow instead of the installation of this
	// connector.
	return []connector.ConnectorTaskTree{
		{
			TaskType:     connector.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []connector.ConnectorTaskTree{},
		},
	}
}
