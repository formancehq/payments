package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ExternalAccount struct {}

func (c *client) GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	// TODO: call PSP to fetch external accounts
    return nil, nil
}
