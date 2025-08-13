package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ListTransactionRequest struct {
	UserID        string
	AccountID     string
	BookedDateGTE time.Time
	BookedDateLTE time.Time
	PageSize      int
	NextPageToken string
}

type ListTransactionResponse struct {
	NextPageToken string        `json:"nextPageToken"`
	Transactions  []Transaction `json:"transactions"`
}

type Amount struct {
	CurrencyCode string `json:"currencyCode"`
	Value        struct {
		Scale string `json:"scale"`
		Value string `json:"unscaledValue"`
	} `json:"value"`
}

type Descriptions struct {
	Detailed struct {
		Unstructured string `json:"unstructured"`
	} `json:"detailed"`
	Display  string `json:"display"`
	Original string `json:"original"`
}

type Types struct {
	FinancialInstitutionTypeCode string `json:"financialInstitutionTypeCode"`
	Type                         string `json:"type"`
}

type Transaction struct {
	ID                  string       `json:"id"`
	AccountID           string       `json:"accountId"`
	Status              string       `json:"status"`
	BookedDateTime      time.Time    `json:"bookedDateTime"`
	TransactionDateTime time.Time    `json:"transactionDateTime"`
	ValueDateTime       time.Time    `json:"valueDateTime"`
	Amount              Amount       `json:"amount"`
	Descriptions        Descriptions `json:"descriptions"`
}

func (c *client) ListTransactions(ctx context.Context, req ListTransactionRequest) (ListTransactionResponse, error) {
	authCode, err := c.getUserAccessToken(ctx, GetUserAccessTokenRequest{
		UserID: req.UserID,
		WantedScopes: []Scopes{
			SCOPES_ACCOUNTS_READ,
			SCOPES_TRANSACTIONS_READ,
			SCOPES_USER_READ,
			SCOPES_CREDENTIALS_READ,
			SCOPES_PROVIDERS_READ,
		},
	})
	if err != nil {
		return ListTransactionResponse{}, err
	}

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	endpoint := fmt.Sprintf("%s/data/v2/transactions", c.endpoint)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListTransactionResponse{}, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authCode))

	query := url.Values{}
	query.Add("accountIdIn", req.AccountID)
	if !req.BookedDateGTE.IsZero() {
		query.Add("bookedDateGte", req.BookedDateGTE.Format(time.DateOnly))
	}
	if !req.BookedDateLTE.IsZero() {
		query.Add("bookedDateLte", req.BookedDateLTE.Format(time.DateOnly))
	}
	query.Add("pageSize", strconv.Itoa(req.PageSize))
	if req.NextPageToken != "" {
		query.Add("pageToken", req.NextPageToken)
	}
	request.URL.RawQuery = query.Encode()

	var response ListTransactionResponse
	_, err = c.httpClient.Do(ctx, request, &response, nil)
	if err != nil {
		return ListTransactionResponse{}, err
	}

	return response, nil
}
