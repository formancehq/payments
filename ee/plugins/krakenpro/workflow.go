package krakenpro

import "github.com/formancehq/payments/pkg/domain/models"

// workflow declares the periodic task tree. See MAPPINGS §3.
//
//	fetch_accounts        (periodic root)
//	fetch_balances        (periodic root — BalanceEx is account-global)
//	fetch_payments        (periodic root)
//	fetch_orders          (periodic root)
//	fetch_conversions     (periodic root)
//
// All tasks are independent roots. fetch_orders is NOT nested under
// fetch_accounts: Kraken can't filter orders by account, so nesting
// would make the engine fan out one identical full-orders fetch per
// account. Orders resolve their wallet refs from the in-memory asset
// cache instead (see orders.go), so they need no accounts dependency.
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
