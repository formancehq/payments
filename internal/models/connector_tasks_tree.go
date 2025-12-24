package models

type TaskType int

const (
	TASK_FETCH_OTHERS TaskType = iota
	TASK_FETCH_ACCOUNTS
	TASK_FETCH_BALANCES
	TASK_FETCH_EXTERNAL_ACCOUNTS
	TASK_FETCH_PAYMENTS
	TASK_FETCH_TRADES
	TASK_CREATE_WEBHOOKS
)

type TaskTreeFetchOther struct{}
type TaskTreeFetchAccounts struct{}
type TaskTreeFetchBalances struct{}
type TaskTreeFetchExternalAccounts struct{}
type TaskTreeFetchPayments struct{}
type TaskTreeFetchTrades struct{}
type TaskTreeCreateWebhooks struct{}

type ConnectorTaskTree struct {
	TaskType     TaskType
	Name         string
	Periodically bool
	NextTasks    []ConnectorTaskTree

	TaskTreeFetchOther            *TaskTreeFetchOther
	TaskTreeFetchAccounts         *TaskTreeFetchAccounts
	TaskTreeFetchBalances         *TaskTreeFetchBalances
	TaskTreeFetchExternalAccounts *TaskTreeFetchExternalAccounts
	TaskTreeFetchPayments         *TaskTreeFetchPayments
	TaskTreeFetchTrades           *TaskTreeFetchTrades
	TaskTreeCreateWebhooks        *TaskTreeCreateWebhooks
}

type ConnectorTasksTree []ConnectorTaskTree
