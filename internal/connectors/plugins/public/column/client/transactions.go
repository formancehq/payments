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

func (c *client) GetTransactions(ctx context.Context, cursor string, pageSize int) ([]*Transaction, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transactions")

	req, err := c.newRequest(ctx, http.MethodGet, "transfers", http.NoBody)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create transactions request: %w", err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(pageSize))
	if cursor != "" {
		q.Add("starting_after", cursor)
	}
	req.URL.RawQuery = q.Encode()

	var res TransactionResponseWrapper[[]*Transaction]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get transactions: %w %w", err, errRes.Error())
	}

	return res.Transfers, res.HasMore, nil
}
