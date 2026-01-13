package kraken

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_EXCHANGE_DATA,
			Name:         "fetch_exchange_data",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
