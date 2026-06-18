package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
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

// GetTransactions returns a page of transactions (newest-first) along with the total
// number of pages for the query, which the caller uses to drain a window oldest-first.
func (c *client) GetTransactions(ctx context.Context, accountID string, page, pageSize int, fromTransactionDate, toTransactionDate time.Time) ([]Transaction, int, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("accounts/%s/transactions", accountID), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create accounts request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("size", strconv.Itoa(pageSize))
	if !fromTransactionDate.IsZero() {
		q.Add("fromTransactionDate", fromTransactionDate.Format("2006-01-02T15:04:05-0700"))
	}
	// The transactions endpoint returns results newest-first and exposes no sort
	// parameter, so we freeze the upper bound of a drain window with toTransactionDate
	// to keep page indices stable while paginating (see fetchNextPayments).
	if !toTransactionDate.IsZero() {
		q.Add("toTransactionDate", toTransactionDate.Format("2006-01-02T15:04:05-0700"))
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]Transaction]
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, 0, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get transactions: %v", errRes.Error()),
			err,
		)
	}
	return res.Content, res.TotalPages, nil
}
