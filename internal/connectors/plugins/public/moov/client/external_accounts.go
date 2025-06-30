package client

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (c *client) GetExternalAccounts(ctx context.Context, accountId string) ([]moov.BankAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	bankAccounts, err := c.service.GetMoovBankAccounts(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("failed to get moov bank accounts: %w", err)
	}

	return bankAccounts, nil
}
