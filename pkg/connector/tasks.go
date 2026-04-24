package connector

import (
	"github.com/formancehq/payments/internal/models"
)

// Task type enum.
type TaskType = models.TaskType

const (
	TASK_FETCH_OTHERS            = models.TASK_FETCH_OTHERS
	TASK_FETCH_ACCOUNTS          = models.TASK_FETCH_ACCOUNTS
	TASK_FETCH_BALANCES          = models.TASK_FETCH_BALANCES
	TASK_FETCH_EXTERNAL_ACCOUNTS = models.TASK_FETCH_EXTERNAL_ACCOUNTS
	TASK_FETCH_PAYMENTS          = models.TASK_FETCH_PAYMENTS
	TASK_CREATE_WEBHOOKS         = models.TASK_CREATE_WEBHOOKS
)

// Task tree types.
type (
	TaskTreeFetchOther            = models.TaskTreeFetchOther
	TaskTreeFetchAccounts         = models.TaskTreeFetchAccounts
	TaskTreeFetchBalances         = models.TaskTreeFetchBalances
	TaskTreeFetchExternalAccounts = models.TaskTreeFetchExternalAccounts
	TaskTreeFetchPayments         = models.TaskTreeFetchPayments
	TaskTreeCreateWebhooks        = models.TaskTreeCreateWebhooks
	ConnectorTaskTree             = models.ConnectorTaskTree
	ConnectorTasksTree            = models.ConnectorTasksTree
)
