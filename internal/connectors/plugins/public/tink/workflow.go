package tink

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
