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

type PayoutRequest struct {
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"` // UMN format: "USD/2", "BTC/8"
	SourceAccountId      string            `json:"sourceAccountId"`
	DestinationAccountId string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type PayoutResponse struct {
	Id                   string            `json:"id"`
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"` // UMN format: "USD/2", "BTC/8"
	SourceAccountId      string            `json:"sourceAccountId"`
	DestinationAccountId string            `json:"destinationAccountId"`
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

	baseURL := c.apiClient.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/payouts", baseURL)

	logging.FromContext(ctx).Debugf("Creating payout: POST %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create payout request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.apiClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute payout request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read payout response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("payout request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var payoutResp PayoutResponse
	if err := json.Unmarshal(respBody, &payoutResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payout response: %w", err)
	}

	logging.FromContext(ctx).Debugf("Payout created: %s with status %s", payoutResp.Id, payoutResp.Status)

	return &payoutResp, nil
}

func (c *client) GetPayoutStatus(ctx context.Context, payoutId string) (*PayoutResponse, error) {
	ctx = metrics.OperationContext(ctx, "get_payout_status")

	baseURL := c.apiClient.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/payouts/%s", baseURL, payoutId)

	logging.FromContext(ctx).Debugf("Getting payout status: GET %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get payout status request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.apiClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get payout status request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read payout status response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get payout status request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var payoutResp PayoutResponse
	if err := json.Unmarshal(respBody, &payoutResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payout status response: %w", err)
	}

	logging.FromContext(ctx).Debugf("Payout status retrieved: %s with status %s", payoutResp.Id, payoutResp.Status)

	return &payoutResp, nil
}