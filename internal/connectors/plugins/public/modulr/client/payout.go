package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "modulr", "initiate_payout")
	// now := time.Now()
	// defer f(ctx, now)

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
	var errRes modulrError
	_, err = c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create payout: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) GetPayout(ctx context.Context, payoutID string) (PayoutResponse, error) {
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "modulr", "get_payout")
	// now := time.Now()
	// defer f(ctx, now)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("payments?id=%s", payoutID), nil)
	if err != nil {
		return PayoutResponse{}, fmt.Errorf("failed to create get payout request: %w", err)
	}

	var res PayoutResponse
	var errRes modulrError
	_, err = c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return PayoutResponse{}, fmt.Errorf("failed to get payout: %w %w", err, errRes.Error())
	}
	return res, nil
}
