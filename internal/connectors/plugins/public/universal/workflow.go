package universal

import "github.com/formancehq/payments/internal/models"

// workflow assembles the install-time ConnectorTasksTree from the
// per-install capability set.
//
//   - FETCH_BALANCES nests under FETCH_ACCOUNTS (balances are
//     account-scoped); dropped if FETCH_ACCOUNTS isn't declared.
//   - FETCH_ORDERS / FETCH_CONVERSIONS sit as siblings — they resolve
//     accounts at runtime via UseAccountLookup. BootstrapOnInstall in
//     plugin.go forces FETCH_ACCOUNTS to complete first.
//   - TASK_CREATE_WEBHOOKS is added when CREATE_WEBHOOKS is declared.
func workflow(declared capabilitySet) models.ConnectorTasksTree {
	tree := models.ConnectorTasksTree{}

	if declared.has(models.CAPABILITY_FETCH_ACCOUNTS) {
		accountsNode := models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		}
		if declared.has(models.CAPABILITY_FETCH_BALANCES) {
			accountsNode.NextTasks = append(accountsNode.NextTasks, models.ConnectorTaskTree{
				TaskType:     models.TASK_FETCH_BALANCES,
				Name:         "fetch_balances",
				Periodically: true,
				NextTasks:    []models.ConnectorTaskTree{},
			})
		}
		tree = append(tree, accountsNode)
	}

	if declared.has(models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
			Name:         "fetch_external_accounts",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	if declared.has(models.CAPABILITY_FETCH_PAYMENTS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	if declared.has(models.CAPABILITY_FETCH_ORDERS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_ORDERS,
			Name:         "fetch_orders",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	if declared.has(models.CAPABILITY_FETCH_CONVERSIONS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_CONVERSIONS,
			Name:         "fetch_conversions",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	if declared.has(models.CAPABILITY_FETCH_OTHERS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_FETCH_OTHERS,
			Name:         "fetch_others",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	if declared.has(models.CAPABILITY_CREATE_WEBHOOKS) {
		tree = append(tree, models.ConnectorTaskTree{
			TaskType:     models.TASK_CREATE_WEBHOOKS,
			Name:         "create_webhooks",
			Periodically: false,
			NextTasks:    []models.ConnectorTaskTree{},
		})
	}

	return tree
}
