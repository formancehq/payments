package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

// Account represents a Bitstamp sub-account with authentication credentials.
// Each account has its own API key/secret pair for accessing its transactions and balances.
type Account struct {
	ID        string `json:"id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	APIKey    string `json:"api_key" validate:"required"`
	ApiSecret string `json:"api_secret" validate:"required"`
}

// GetAccounts returns the list of accounts configured in the connector.
// Unlike most PSP APIs, Bitstamp accounts are defined at setup time in the config,
// not fetched from an API. This method returns only the ID and Name for security
// (API credentials are excluded from the response).
func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	_ = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	// put accounts from config in the response, but only the id and name fields
	accounts := make([]*Account, len(c.accounts))
	for i, account := range c.accounts {
		accounts[i] = &Account{
			ID:   account.ID,
			Name: account.Name,
		}
	}

	return accounts, nil
}
