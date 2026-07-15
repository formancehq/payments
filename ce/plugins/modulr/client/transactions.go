package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/pkg/domain/metrics"
	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
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
func (c *client) GetTransactions(ctx context.Context, accountID string, page, pageSize int, fromPostedDate, toPostedDate time.Time) ([]Transaction, int, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("accounts/%s/transactions", accountID), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create accounts request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("size", strconv.Itoa(pageSize))
	if !fromPostedDate.IsZero() {
		q.Add("fromPostedDate", fromPostedDate.Format(transactionFilterLayout))
	}
	// The transactions endpoint returns results newest-first by postedDate (verified
	// against the sandbox: postedDate is strictly descending, transactionDate is not)
	// and exposes no sort parameter, so we freeze the upper bound of a drain window
	// with toPostedDate to keep page indices stable while paginating (see
	// fetchNextPayments).
	if !toPostedDate.IsZero() {
		q.Add("toPostedDate", formatToPostedDate(toPostedDate))
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

// transactionFilterLayout is the second-precision layout the transactions endpoint accepts
// for its fromPostedDate / toPostedDate filters.
const transactionFilterLayout = "2006-01-02T15:04:05-0700"

// formatToPostedDate formats a drain-window ceiling for the toPostedDate filter.
// The filter is whole-second and INCLUSIVE: toPostedDate=2017-01-28T01:01:01+0000
// returns transactions through 2017-01-28T01:01:01.999 (verified against the Modulr
// sandbox). So we truncate the ceiling (which carries milliseconds) down to its second —
// the whole ceiling second, including the ceiling transaction itself, is still returned.
// We deliberately do NOT round up: that would widen the window into the next second and
// admit transactions newer than the ceiling, shifting the newest-first page boundaries
// mid-drain. fetchNextPayments re-applies the exact ceiling client-side.
func formatToPostedDate(toPostedDate time.Time) string {
	return toPostedDate.Truncate(time.Second).Format(transactionFilterLayout)
}
