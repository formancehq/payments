package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {
	ID                    string `json:"id"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
	CompletedAt           string `json:"completed_at"`
	Status                string `json:"status"`
	Type                  string `json:"type"`
	Amount                int64  `json:"amount"`
	CurrencyCode          string `json:"currency_code"`
	IsIncoming            bool   `json:"is_incoming"`
	IdempotencyKey        string `json:"idempotency_key"`
	Description           string `json:"description"`
	SenderInternalAccount struct {
		BankAccountID   string `json:"bank_account_id"`
		AccountNumberID string `json:"account_number_id"`
	} `json:"sender_internal_account"`
	ExternalSource struct {
		BankName       string `json:"bank_name"`
		SenderName     string `json:"sender_name"`
		CounterpartyID string `json:"counterparty_id"`
	} `json:"external_source"`
	ReceiverInternalAccount struct {
		BankAccountID   string `json:"bank_account_id"`
		AccountNumberID string `json:"account_number_id"`
	} `json:"receiver_internal_account"`
	ExternalDestination struct {
		CounterpartyID string `json:"counterparty_id"`
	} `json:"external_destination"`
}

type TransactionResponseWrapper[t any] struct {
	Transfers t    `json:"transfers"`
	HasMore   bool `json:"has_more"`
}

func (c *client) GetTransactions(ctx context.Context, t Timeline, pageSize int) (results []*Transaction, timeline Timeline, hasMore bool, err error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")
	var endpoint = "transfers"

	timeline = t
	results = make([]*Transaction, 0, pageSize)
	if !timeline.IsCaughtUp() {
		var oldest *Transaction
		oldest, timeline, hasMore, err = c.scanForOldest(ctx, timeline, endpoint, pageSize)
		if err != nil {
			return results, timeline, false, err
		}
		// either there are no records or we haven't found the start yet
		if !timeline.IsCaughtUp() {
			return results, timeline, hasMore, nil
		}
		results = append(results, oldest)
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if timeline.LatestID != "" {
		q.Add("ending_before", timeline.LatestID)
	}
	req.URL.RawQuery = q.Encode()

	var res TransactionResponseWrapper[[]*Transaction]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, timeline, false, fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}

	transactions := reverseTransactions(res.Transfers)
	results = append(results, transactions...)
	if len(results) > 0 {
		timeline.LatestID = results[len(results)-1].ID
	}
	return results, timeline, res.HasMore, nil
}

func reverseTransactions(in []*Transaction) []*Transaction {
	out := make([]*Transaction, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
