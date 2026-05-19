package bitstamp

import "github.com/formancehq/payments/internal/models"

// workflow declares the periodic task tree for the connector. The
// shape follows MAPPINGS.md §2.1 and the convention emerging from
// PR #679 review on the original Bitstamp connector + PR #707 on
// Coinbase Prime:
//
//	fetch_accounts (periodic)
//	  └── fetch_balances (FromPayload — no extra API call)
//
//	fetch_payments     (periodic root)
//	fetch_orders       (periodic root)
//	fetch_conversions  (periodic root)
//
// fetch_balances is nested under fetch_accounts (not periodic itself)
// so the per-account balance is derived from PSPAccount.Raw already
// returned by the accounts task — Qonto pattern. This eliminates a
// second /api/v2/account_balances/ call per cycle.
//
// payments / orders / conversions are independent roots because none
// of them require a parent account ID (Bitstamp's endpoints are
// account-global at the API-key level).
func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:  models.TASK_FETCH_BALANCES,
					Name:      "fetch_balances",
					NextTasks: []models.ConnectorTaskTree{},
				},
			},
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
