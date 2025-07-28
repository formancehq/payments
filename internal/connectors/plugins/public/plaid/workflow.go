package plaid

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	// Do not launch fetch data workflows here, since we're depending on the
	// users to finish the link flow instead of the installation of this
	// connector.
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
