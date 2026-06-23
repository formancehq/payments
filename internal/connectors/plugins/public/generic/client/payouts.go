package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/domain/metrics"
)

type PayoutRequest struct {
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountID      string            `json:"sourceAccountId"`
	DestinationAccountID string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type PayoutResponse struct {
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

func (c *client) CreatePayout(ctx context.Context, request *PayoutRequest) (*PayoutResponse, error) {
	ctx = metrics.OperationContext(ctx, "create_payout")

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payout request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/payouts", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create payout request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var resp PayoutResponse
	var errResp genericAPIError
	if _, err = c.httpClient.Do(ctx, req, &resp, &errResp); err != nil {
		return nil, fmt.Errorf("failed to create payout: %w", err)
	}
	return &resp, nil
}
