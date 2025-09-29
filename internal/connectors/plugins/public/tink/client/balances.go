package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type AccountBalanceResponse struct {
	AccountId string          `json:"accountId"`
	Refreshed time.Time       `json:"refreshed"`
	Balances  AccountBalances `json:"balances"`
}

type AccountBalances struct {
	Booked      AccountBalance `json:"booked"`
	Available   AccountBalance `json:"available"`
	CreditLimit AccountBalance `json:"creditLimit"`
}
type AccountBalance struct {
	Value            AccountBalanceValue `json:"value"`
	CurrencyCode     string              `json:"currencyCode"`
	ValueInMinorUnit json.Number         `json:"valueInMinorUnit"`
}

type AccountBalanceValue struct {
	Scale         json.Number `json:"scale"`
	UnscaledValue json.Number `json:"unscaledValue"`
}

func (c *client) GetAccountBalances(ctx context.Context, userID string, accountID string) (AccountBalanceResponse, error) {
	authCode, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: userID,
		WantedScopes: []Scopes{
			SCOPES_ACCOUNTS_READ,
			SCOPES_TRANSACTIONS_READ,
			SCOPES_BALANCES_READ,
			SCOPES_USER_READ,
		},
	})
	if err != nil {
		return AccountBalanceResponse{}, err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_balance")

	endpoint := fmt.Sprintf("%s/data/v2/accounts/%s/balances", c.endpoint, accountID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AccountBalanceResponse{}, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	var response AccountBalanceResponse
	_, err = c.userClient.Do(ctx, request, &response, nil)
	if err != nil {
		return AccountBalanceResponse{}, err
	}

	return response, nil
}
