package models_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestToV3Provider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		provider string
		expected string
	}{
		{"STRIPE", "stripe"},
		{"MODULR", "modulr"},
		{"CURRENCY-CLOUD", "currencycloud"},
		{"WISE", "wise"},
		{"MANGOPAY", "mangopay"},
		{"BANKING-CIRCLE", "bankingcircle"},
		{"ADYEN", "adyen"},
		{"ATLAR", "atlar"},
		{"DUMMY-PAY", "dummypay"},
		{"GENERIC", "generic"},
		{"MONEYCORP", "moneycorp"},
		{"unknown", "unknown"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			result := models.ToV3Provider(tc.provider)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConnectorTaskTree(t *testing.T) {
	t.Parallel()

	tree := models.ConnectorTaskTree{}
	assert.Empty(t, tree.NextTasks)
	assert.Empty(t, tree.Name)

	tree = models.ConnectorTaskTree{
		TaskType: models.TASK_FETCH_ACCOUNTS,
		Name:     "fetch-accounts",
	}
	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tree.TaskType)
	assert.Equal(t, "fetch-accounts", tree.Name)
	assert.Empty(t, tree.NextTasks)

	childTask := models.ConnectorTaskTree{
		TaskType: models.TASK_FETCH_BALANCES,
		Name:     "fetch-balances",
	}
	tree = models.ConnectorTaskTree{
		TaskType:  models.TASK_FETCH_ACCOUNTS,
		Name:      "fetch-accounts",
		NextTasks: []models.ConnectorTaskTree{childTask},
	}
	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tree.TaskType)
	assert.Equal(t, "fetch-accounts", tree.Name)
	assert.Len(t, tree.NextTasks, 1)
	assert.Equal(t, models.TASK_FETCH_BALANCES, tree.NextTasks[0].TaskType)
	assert.Equal(t, "fetch-balances", tree.NextTasks[0].Name)
}

func TestConnectorTasksTree(t *testing.T) {
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
	tasksTree = models.ConnectorTasksTree{task1, task2}
	assert.Len(t, tasksTree, 2)
	assert.Equal(t, models.TASK_FETCH_ACCOUNTS, tasksTree[0].TaskType)
	assert.Equal(t, "fetch-accounts", tasksTree[0].Name)
	assert.Equal(t, models.TASK_FETCH_BALANCES, tasksTree[1].TaskType)
	assert.Equal(t, "fetch-balances", tasksTree[1].Name)
}

func TestPSPWebhookConfig(t *testing.T) {
	t.Parallel()

	config := models.PSPWebhookConfig{
		Name:    "test-webhook",
		URLPath: "/webhook",
	}
	assert.Equal(t, "test-webhook", config.Name)
	assert.Equal(t, "/webhook", config.URLPath)
}

func TestPSPWebhook(t *testing.T) {
	t.Parallel()

	basicAuth := &models.BasicAuth{
		Username: "user",
		Password: "pass",
	}
	webhook := models.PSPWebhook{
		BasicAuth: basicAuth,
		QueryValues: map[string][]string{
			"key": {"value"},
		},
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: []byte(`{"key": "value"}`),
	}
	assert.Equal(t, basicAuth, webhook.BasicAuth)
	assert.Equal(t, "value", webhook.QueryValues["key"][0])
	assert.Equal(t, "application/json", webhook.Headers["Content-Type"][0])
	assert.Equal(t, []byte(`{"key": "value"}`), webhook.Body)
}

func TestPSPOther(t *testing.T) {
	t.Parallel()

	other := models.PSPOther{
		ID:    "test-other",
		Other: json.RawMessage(`{"key": "value"}`),
	}
	assert.Equal(t, "test-other", other.ID)
	assert.Equal(t, json.RawMessage(`{"key": "value"}`), other.Other)
}

func TestPSPBalance(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	amount := big.NewInt(100)
	balance := models.PSPBalance{
		AccountReference: "test-account",
		CreatedAt:        now,
		Asset:            "USD/2",
		Amount:           amount,
	}
	assert.Equal(t, "test-account", balance.AccountReference)
	assert.Equal(t, now, balance.CreatedAt)
	assert.Equal(t, "USD/2", balance.Asset)
	assert.Equal(t, amount, balance.Amount)
}
