package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	AccountReference string `json:"accountReference"`
	Asset            string `json:"asset"`
	AmountInMinors   int64  `json:"amountInMinors"`
	ReportedAt       string `json:"reportedAt"`
	ImportedAt       string `json:"importedAt"`
	UpdatedAt        string `json:"updatedAt"`
}

func (c *client) GetAccountBalances(ctx context.Context, cursor string, pageSize int) ([]Balance, bool, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	// TODO: call PSP to have balances
	return nil, false, "", nil
}
