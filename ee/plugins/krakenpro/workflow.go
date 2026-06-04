package krakenpro

import "github.com/formancehq/payments/internal/models"

// workflow declares the periodic task tree. See MAPPINGS §3.
//
//	fetch_accounts        (periodic root)
//	  └─ fetch_orders     (periodic; reads wallet refs via AccountLookup)
//	fetch_balances        (periodic root — BalanceEx is account-global)
//	fetch_payments        (periodic root)
//	fetch_conversions     (periodic root)
//
// Orders sit nested under accounts so the engine reads the accounts
// table fresh before each order cycle. BootstrapOnInstall in
// plugin.go additionally drains a full FETCH_ACCOUNTS pass on
// install so the very first FETCH_ORDERS cycle never sees an empty
// accounts table.
func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_ORDERS,
					Name:         "fetch_orders",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
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
			TaskType:     models.TASK_FETCH_CONVERSIONS,
			Name:         "fetch_conversions",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
