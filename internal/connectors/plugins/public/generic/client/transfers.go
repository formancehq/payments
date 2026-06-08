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
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountID      string            `json:"sourceAccountId"`
	DestinationAccountID string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type TransferResponse struct {
	Id                   string            `json:"id"`
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountID      string            `json:"sourceAccountId"`
	DestinationAccountID string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Status               string            `json:"status"`
	CreatedAt            string            `json:"createdAt"`
	UpdatedAt            *string           `json:"updatedAt,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

func (c *client) CreateTransfer(ctx context.Context, request *TransferRequest) (*TransferResponse, error) {
	ctx = metrics.OperationContext(ctx, "create_transfer")

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/transfers", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var resp TransferResponse
	var errResp genericAPIError
	if _, err = c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
		return nil, fmt.Errorf("failed to create transfer: %w", err)
	}
	return &resp, nil
}
