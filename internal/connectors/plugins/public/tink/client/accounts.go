package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ListAccountsResponse struct {
	Accounts      []Account `json:"accounts"`
	NextPageToken string    `json:"nextPageToken"`
}

type AccountBalances struct {
	Booked      AccountBalance `json:"booked"`
	Available   AccountBalance `json:"available"`
	CreditLimit AccountBalance `json:"creditLimit"`
}
type AccountBalance struct {
	Amount Amount `json:"amount"`
}

type AccountBalanceValue struct {
	Scale         json.Number `json:"scale"`
	UnscaledValue json.Number `json:"unscaledValue"`
}

type Dates struct {
	LastRefreshed time.Time `json:"lastRefreshed"`
}

type Account struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Balances AccountBalances `json:"balances"`
	Dates    Dates           `json:"dates"`
}

func (c *client) ListAccounts(ctx context.Context, userID string, nextPageToken string) (ListAccountsResponse, error) {
	authCode, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: userID,
		WantedScopes: []Scopes{
			SCOPES_ACCOUNTS_READ,
			SCOPES_TRANSACTIONS_READ,
			SCOPES_USER_READ,
			SCOPES_CREDENTIALS_READ,
			SCOPES_PROVIDERS_READ,
		},
	})
	if err != nil {
		return ListAccountsResponse{}, err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	endpoint := fmt.Sprintf("%s/data/v2/accounts", c.endpoint)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListAccountsResponse{}, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	query := url.Values{}
	if nextPageToken != "" {
		query.Add("pageToken", nextPageToken)
	}
	request.URL.RawQuery = query.Encode()

	var response ListAccountsResponse
	_, err = c.userClient.Do(ctx, request, &response, nil)
	if err != nil {
		return ListAccountsResponse{}, err
	}

	return response, nil
}

func (c *client) GetAccount(ctx context.Context, userID string, accountID string) (Account, error) {
	authCode, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: userID,
		WantedScopes: []Scopes{
			SCOPES_ACCOUNTS_READ,
		},
	})
	if err != nil {
		return Account{}, err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_account")

	endpoint := fmt.Sprintf("%s/data/v2/accounts/%s", c.endpoint, accountID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Account{}, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	var response Account
	_, err = c.userClient.Do(ctx, request, &response, nil)
	if err != nil {
		return Account{}, err
	}

	return response, nil
}
