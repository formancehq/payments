package teller

import "github.com/formancehq/payments/internal/models"

func workflow() models.ConnectorTasksTree {
	// Teller has no webhook API and no periodic polling for the prototype.
	// Data fetching is triggered by open banking link completion.
	return []models.ConnectorTaskTree{}
}
