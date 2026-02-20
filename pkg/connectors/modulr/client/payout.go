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

type PayoutRequest struct {
	IdempotencyKey    string      `json:"-"`
	SourceAccountID   string      `json:"sourceAccountId"`
	Destination       Destination `json:"destination"`
	Currency          string      `json:"currency"`
	Amount            json.Number `json:"amount"`
	Reference         string      `json:"reference"`
	ExternalReference string      `json:"externalReference"`
}

type PayoutResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	CreatedDate       string  `json:"createdDate"`
	ExternalReference string  `json:"externalReference"`
	ApprovalStatus    string  `json:"approvalStatus"`
	Message           string  `json:"message"`
	Details           Details `json:"details"`
}

func (c *client) InitiatePayout(ctx context.Context, payoutRequest *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	body, err := json.Marshal(payoutRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildEndpoint("payments"), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create payout request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-mod-nonce", payoutRequest.IdempotencyKey)

	var res PayoutResponse
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to create payout: %v", errRes.Error()),
			err,
		)
	}
	return &res, nil
}

func (c *client) GetPayout(ctx context.Context, payoutID string) (PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_payout")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("payments?id=%s", payoutID), nil)
	if err != nil {
		return PayoutResponse{}, fmt.Errorf("failed to create get payout request: %w", err)
	}

	var res PayoutResponse
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return PayoutResponse{}, connector.NewWrappedError(
			fmt.Errorf("failed to get payout: %v", errRes.Error()),
			err,
		)
	}
	return res, nil
}
