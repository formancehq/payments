package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ListAccountsResponse struct {
	Accounts      []Account `json:"accounts"`
	NextPageToken string    `json:"nextPageToken"`
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
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
	_, err = c.httpClient.Do(ctx, request, &response, nil)
	if err != nil {
		return ListAccountsResponse{}, err
	}

	return response, nil
}
