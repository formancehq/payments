package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type Counterparties struct {
	ID                   string  `json:"id"`
	AccountNumber        string  `json:"account_number"`
	AccountType          string  `json:"account_type"`
	Address              Address `json:"address,omitempty"`
	CreatedAt            string  `json:"created_at"`
	Description          string  `json:"description"`
	Email                string  `json:"email"`
	IsColumnAccount      bool    `json:"is_column_account"`
	LegalID              string  `json:"legal_id"`
	LegalType            string  `json:"legal_type"`
	LocalAccountNumber   string  `json:"local_account_number"`
	LocalBankCode        string  `json:"local_bank_code"`
	LocalBankCountryCode string  `json:"local_bank_country_code"`
	LocalBankName        string  `json:"local_bank_name"`
	Name                 string  `json:"name"`
	Phone                string  `json:"phone"`
	RoutingNumber        string  `json:"routing_number"`
	RoutingNumberType    string  `json:"routing_number_type"`
	UpdatedAt            string  `json:"updated_at"`
	Wire                 Wire    `json:"wire"`
	WireDrawdownAllowed  bool    `json:"wire_drawdown_allowed"`
}

type CounterpartiesResponseWrapper[t any] struct {
	Counterparties t    `json:"counterparties"`
	HasMore        bool `json:"has_more"`
}

func (c *client) GetCounterparties(ctx context.Context, cursor string, pageSize int) ([]*Counterparties, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	req, err := c.newRequest(ctx, http.MethodGet, "counterparties", http.NoBody)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create external account request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("starting_after", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res CounterpartiesResponseWrapper[[]*Counterparties]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get external accounts: %w %w", err, errRes.Error())
	}
	return res.Counterparties, res.HasMore, nil
}
