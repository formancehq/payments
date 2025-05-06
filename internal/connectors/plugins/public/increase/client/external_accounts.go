package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ExternalAccount struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	RoutingNumber string `json:"routing_number"`
	AccountNumber string `json:"account_number"`
	Status        string `json:"status"`
	Type          string `json:"type"`
	AccountHolder string `json:"account_holder"`
	CreatedAt     string `json:"created_at"`
}

func (c *client) GetExternalAccounts(ctx context.Context, pageSize int, cursor string) ([]*ExternalAccount, string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	req, err := c.newRequest(ctx, http.MethodGet, "external_accounts", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create external account request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res ResponseWrapper[[]*ExternalAccount]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get external accounts: %w %w", err, errRes.Error())
	}
	return res.Data, res.NextCursor, nil
}
