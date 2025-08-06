package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

func getBankAccountCacheKey(accessToken string, bankAccountID int) string {
	return fmt.Sprintf("%s:%d", accessToken, bankAccountID)
}

type Currency struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Precision int    `json:"precision"`
}

type BankAccount struct {
	ID           int      `json:"id"`
	ConnectionID int      `json:"id_connection"`
	Currency     Currency `json:"currency"`
	OriginalName string   `json:"original_name"`
}

func (c *client) GetBankAccount(ctx context.Context, accessToken string, bankAccountID int) (BankAccount, error) {
	cacheKey := getBankAccountCacheKey(accessToken, bankAccountID)

	if bankAccount, ok := c.bankAccountsCache.Get(cacheKey); ok {
		return bankAccount, nil
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_bank_account")

	endpoint := fmt.Sprintf("%s/2.0/users/me/accounts/%d", c.endpoint, bankAccountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return BankAccount{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	var resp BankAccount
	var errResp powensError
	if _, err := c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
		return BankAccount{}, fmt.Errorf("failed to get bank account: %w", errResp.Error())
	}

	c.bankAccountsCache.Add(cacheKey, resp)

	return resp, nil
}
