package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type transferRequest struct {
	Transfer struct {
		Attributes *TransferRequest `json:"attributes"`
	} `json:"data"`
}

type TransferRequest struct {
	SourceAccountID    string      `json:"-"`
	IdempotencyKey     string      `json:"-"`
	ReceivingAccountID string      `json:"receivingAccountId"`
	TransferAmount     json.Number `json:"transferAmount"`
	TransferCurrency   string      `json:"transferCurrency"`
	TransferReference  string      `json:"transferReference,omitempty"`
	ClientReference    string      `json:"clientReference,omitempty"`
}

type transferResponse struct {
	Transfer *TransferResponse `json:"data"`
}

type TransferAttributes struct {
	SendingAccountID     int64       `json:"sendingAccountId"`
	SendingAccountName   string      `json:"sendingAccountName"`
	ReceivingAccountID   int64       `json:"receivingAccountId"`
	ReceivingAccountName string      `json:"receivingAccountName"`
	CreatedAt            string      `json:"createdAt"`
	CreatedBy            string      `json:"createdBy"`
	UpdatedAt            string      `json:"updatedAt"`
	TransferReference    string      `json:"transferReference"`
	ClientReference      string      `json:"clientReference"`
	TransferDate         string      `json:"transferDate"`
	TransferAmount       json.Number `json:"transferAmount"`
	TransferCurrency     string      `json:"transferCurrency"`
	TransferStatus       string      `json:"transferStatus"`
}

type TransferResponse struct {
	ID         string             `json:"id"`
	Attributes TransferAttributes `json:"attributes"`
}

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	endpoint := fmt.Sprintf("%s/accounts/%s/transfers", c.endpoint, tr.SourceAccountID)

	reqBody := &transferRequest{}
	reqBody.Transfer.Attributes = tr
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", tr.IdempotencyKey)

	var transferResponse transferResponse
	var errRes moneycorpErrors
	_, err = c.httpClient.Do(ctx, req, &transferResponse, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to initiate transfer: %v", errRes.Error()),
			err,
		)
	}

	return transferResponse.Transfer, nil
}

func (c *client) GetTransfer(ctx context.Context, accountID string, transferID string) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_transfer")

	endpoint := fmt.Sprintf("%s/accounts/%s/transfers/%s", c.endpoint, accountID, transferID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create get transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var transferResponse transferResponse
	var errRes moneycorpErrors
	_, err = c.httpClient.Do(ctx, req, &transferResponse, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to get transfer: %v", errRes.Error()),
			err,
		)
	}

	return transferResponse.Transfer, nil
}
