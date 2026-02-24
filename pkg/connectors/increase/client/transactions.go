package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type Source struct {
	DestinationAccountID  string `json:"destination_account_id"`
	SourceAccountID       string `json:"source_account_id"`
	Category              string `json:"category"`
	TransferID            string `json:"transfer_id"`
	WireTransferID        string `json:"wire_transfer_id"`
	InboundCheckDepositID string `json:"inbound_check_deposit_id"`
	CheckDepositID        string `json:"check_deposit_id"`
	InboundAchTransferID  string `json:"inbound_ach_transfer_id"`
	InboundWireTransferID string `json:"inbound_wire_transfer_id"`
	ID                    string `json:"id"`
	Amount                int64  `json:"amount"`
}

type Transaction struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`
	Date        string `json:"date"`
	Description string `json:"description"`
	RouteID     string `json:"route_id"`
	RouteType   string `json:"route_type"`
	Type        string `json:"type"`
	Source      Source `json:"source"`
}

func (c *client) GetTransactions(ctx context.Context, pageSize int, t Timeline) (results []*Transaction, timeline Timeline, hasMore bool, err error) {
	return c.getTransactionsByType(ctx, "transactions", pageSize, t)
}

func (c *client) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}

func (c *client) GetPendingTransactions(ctx context.Context, pageSize int, t Timeline) (results []*Transaction, timeline Timeline, hasMore bool, err error) {
	return c.getTransactionsByType(ctx, "pending_transactions", pageSize, t)
}

func (c *client) GetPendingTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_pending_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("pending_transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending transaction request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}

func (c *client) GetDeclinedTransactions(ctx context.Context, pageSize int, t Timeline) (results []*Transaction, timeline Timeline, hasMore bool, err error) {
	return c.getTransactionsByType(ctx, "declined_transactions", pageSize, t)
}

func (c *client) getTransactionsByType(ctx context.Context, endpoint string, pageSize int, t Timeline) (results []*Transaction, timeline Timeline, hasMore bool, err error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, fmt.Sprintf("list_%s", endpoint))
	timeline = t
	results = make([]*Transaction, 0, pageSize)

	// First phase: scroll back in time to find the oldest record
	if !timeline.IsCaughtUp() {
		var oldest []*Transaction
		oldest, timeline, hasMore, err = c.scanForOldest(ctx, timeline, endpoint, pageSize)
		if err != nil {
			return results, timeline, false, err
		}
		// If we got data back, this is our oldest data
		if len(oldest) > 0 {
			results = reverseTransactions(oldest)
			return results, timeline, hasMore, nil
		}
		// If we haven't found the start yet, continue scanning
		if !timeline.IsCaughtUp() {
			return results, timeline, hasMore, nil
		}
	}

	// Second phase: fetch forward using stored cursors
	var res ResponseWrapper[[]*Transaction]
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))

	// If we have stored cursors, use the last one
	if len(timeline.Cursors) > 0 {
		// Get the last cursor and remove it from the list
		cursor := timeline.Cursors[len(timeline.Cursors)-1]
		timeline.Cursors = timeline.Cursors[:len(timeline.Cursors)-1]
		q.Add("cursor", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}

	transactions := reverseTransactions(res.Data)
	results = append(results, transactions...)

	// We have more data if we have more cursors or if there's a next cursor
	hasMore = len(timeline.Cursors) > 0 || q.Get("cursor") != ""

	return results, timeline, hasMore, nil
}

func (c *client) GetDeclinedTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_declined_transaction")

	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("declined_transactions/%s", transactionID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create declined transaction request: %w", err)
	}

	var res Transaction
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get declined transaction: %w %w", err, errRes.Error())
	}

	return &res, nil
}

// Increase returns data in reverse chronological order so we need to reverse the slice
func reverseTransactions(in []*Transaction) []*Transaction {
	out := make([]*Transaction, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
