package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

//nolint:tagliatelle // allow different styled tags in client
type Transaction struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	Amount          json.Number `json:"amount"`
	Credit          bool        `json:"credit"`
	SourceID        string      `json:"sourceId"`
	Description     string      `json:"description"`
	PostedDate      string      `json:"postedDate"`
	TransactionDate string      `json:"transactionDate"`
	Account         Account     `json:"account"`
	AdditionalInfo  interface{} `json:"additionalInfo"`
}

func (c *client) GetTransactions(ctx context.Context, accountID string, page, pageSize int, fromTransactionDate time.Time) ([]Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("accounts/%s/transactions", accountID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("size", strconv.Itoa(pageSize))
	if !fromTransactionDate.IsZero() {
		q.Add("fromTransactionDate", fromTransactionDate.Format("2006-01-02T15:04:05-0700"))
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]Transaction]
	var errRes modulrError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}
	return res.Content, nil
}
