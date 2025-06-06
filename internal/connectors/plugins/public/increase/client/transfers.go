package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type TransferRequest struct {
	AccountID            string `json:"account_id"`
	Amount               int64  `json:"amount"`
	Description          string `json:"description"`
	DestinationAccountID string `json:"destination_account_id"`
}

type TransferResponse struct {
	ID                       string `json:"id"`
	AccountID                string `json:"account_id"`
	Amount                   int64  `json:"amount"`
	Currency                 string `json:"currency"`
	DestinationAccountID     string `json:"destination_account_id"`
	DestinationTransactionID string `json:"destination_transaction_id"`
	TransactionID            string `json:"transaction_id"`
	PendingTransactionID     string `json:"pending_transaction_id"`
	Description              string `json:"description"`
	Status                   string `json:"status"`
	CreatedAt                string `json:"created_at"`
}

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest, idempotencyKey string) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	body, err := json.Marshal(tr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "account_transfers", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res TransferResponse
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate transfer: %w %w", err, errRes.Error())
	}
	return &res, nil
}
