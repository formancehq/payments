package bitstamp

import "github.com/formancehq/payments/internal/models"

// workflow declares the periodic task tree. See MAPPINGS §3.
//
//	fetch_accounts     (periodic root)
//	fetch_balances     (periodic root)
//	fetch_payments     (periodic root)
//	fetch_orders       (periodic root)
//	fetch_conversions  (periodic root)
//
// Payments/orders/conversions are independent roots — Bitstamp
// endpoints are account-global at the API-key level so no parent
// context is needed.
func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_BALANCES,
			Name:         "fetch_balances",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_ORDERS,
			Name:         "fetch_orders",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
		{
			TaskType:     models.TASK_FETCH_CONVERSIONS,
			Name:         "fetch_conversions",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
