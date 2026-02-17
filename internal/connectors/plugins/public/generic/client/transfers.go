package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/logging"
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

	baseURL := c.apiClient.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/transfers", baseURL)

	logging.FromContext(ctx).Debugf("Creating transfer: POST %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.apiClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transfer request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read transfer response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("transfer request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var transferResp TransferResponse
	if err := json.Unmarshal(respBody, &transferResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transfer response: %w", err)
	}

	logging.FromContext(ctx).Debugf("Transfer created: %s with status %s", transferResp.Id, transferResp.Status)

	return &transferResp, nil
}
