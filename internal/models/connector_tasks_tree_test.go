package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestTaskType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, models.TaskType(0), models.TASK_FETCH_OTHERS)
	assert.Equal(t, models.TaskType(1), models.TASK_FETCH_ACCOUNTS)
	assert.Equal(t, models.TaskType(2), models.TASK_FETCH_BALANCES)
	assert.Equal(t, models.TaskType(3), models.TASK_FETCH_EXTERNAL_ACCOUNTS)
	assert.Equal(t, models.TaskType(4), models.TASK_FETCH_PAYMENTS)
	assert.Equal(t, models.TaskType(5), models.TASK_CREATE_WEBHOOKS)
}

func TestTaskTreeStructs(t *testing.T) {
	t.Parallel()

	fetchOther := &models.TaskTreeFetchOther{}
	fetchAccounts := &models.TaskTreeFetchAccounts{}
	fetchBalances := &models.TaskTreeFetchBalances{}
	fetchExternalAccounts := &models.TaskTreeFetchExternalAccounts{}
	fetchPayments := &models.TaskTreeFetchPayments{}
	createWebhooks := &models.TaskTreeCreateWebhooks{}

	tree := models.ConnectorTaskTree{
		TaskType:                      models.TASK_FETCH_ACCOUNTS,
		Name:                          "fetch-accounts",
		Periodically:                  true,
		TaskTreeFetchAccounts:         fetchAccounts,
		TaskTreeFetchBalances:         fetchBalances,
		TaskTreeFetchExternalAccounts: fetchExternalAccounts,
		TaskTreeFetchOther:            fetchOther,
		TaskTreeFetchPayments:         fetchPayments,
		TaskTreeCreateWebhooks:        createWebhooks,
	}

	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tree.TaskType)
	assert.Equal(t, "fetch-accounts", tree.Name)
	assert.True(t, tree.Periodically)
	assert.Equal(t, fetchAccounts, tree.TaskTreeFetchAccounts)
	assert.Equal(t, fetchBalances, tree.TaskTreeFetchBalances)
	assert.Equal(t, fetchExternalAccounts, tree.TaskTreeFetchExternalAccounts)
	assert.Equal(t, fetchOther, tree.TaskTreeFetchOther)
	assert.Equal(t, fetchPayments, tree.TaskTreeFetchPayments)
	assert.Equal(t, createWebhooks, tree.TaskTreeCreateWebhooks)
}

func TestConnectorTaskTreeWithNextTasks(t *testing.T) {
	t.Parallel()

	balancesTask := models.ConnectorTaskTree{
		TaskType:              models.TASK_FETCH_BALANCES,
		Name:                  "fetch-balances",
		TaskTreeFetchBalances: &models.TaskTreeFetchBalances{},
	}

	paymentsTask := models.ConnectorTaskTree{
		TaskType:              models.TASK_FETCH_PAYMENTS,
		Name:                  "fetch-payments",
		TaskTreeFetchPayments: &models.TaskTreeFetchPayments{},
	}

	accountsTask := models.ConnectorTaskTree{
		TaskType:              models.TASK_FETCH_ACCOUNTS,
		Name:                  "fetch-accounts",
		TaskTreeFetchAccounts: &models.TaskTreeFetchAccounts{},
		NextTasks:             []models.ConnectorTaskTree{balancesTask, paymentsTask},
	}

	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, accountsTask.TaskType)
	assert.Equal(t, "fetch-accounts", accountsTask.Name)
	assert.Len(t, accountsTask.NextTasks, 2)
	
	assert.Equal(t, models.TASK_FETCH_BALANCES, accountsTask.NextTasks[0].TaskType)
	assert.Equal(t, "fetch-balances", accountsTask.NextTasks[0].Name)
	
	assert.Equal(t, models.TASK_FETCH_PAYMENTS, accountsTask.NextTasks[1].TaskType)
	assert.Equal(t, "fetch-payments", accountsTask.NextTasks[1].Name)
}

func TestConnectorTasksTreeSlice(t *testing.T) {
	t.Parallel()

	tasksTree := models.ConnectorTasksTree{}
	assert.Empty(t, tasksTree)

	task1 := models.ConnectorTaskTree{
		TaskType: models.TASK_FETCH_ACCOUNTS,
		Name:     "fetch-accounts",
	}
	
	task2 := models.ConnectorTaskTree{
		TaskType: models.TASK_FETCH_BALANCES,
		Name:     "fetch-balances",
	}
	
	tasksTree = append(tasksTree, task1)
	assert.Len(t, tasksTree, 1)
	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tasksTree[0].TaskType)
	
	tasksTree = append(tasksTree, task2)
	assert.Len(t, tasksTree, 2)
	assert.Equal(t, models.TASK_FETCH_BALANCES, tasksTree[1].TaskType)
	
	tasksTree2 := models.ConnectorTasksTree{task1, task2}
	assert.Len(t, tasksTree2, 2)
	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tasksTree2[0].TaskType)
	assert.Equal(t, models.TASK_FETCH_BALANCES, tasksTree2[1].TaskType)
}
